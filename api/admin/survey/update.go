package survey

import (
	"reflect"
	"runtime"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/cache"
	"app/dao/model"
	"app/dao/query"
	"app/dao/repo"
	"app/schema"
)

// UpdateHandler API router注册点
func UpdateHandler() gin.HandlerFunc {
	api := UpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdate).Pointer()).Name()] = api
	return hfUpdate
}

type UpdateApi struct {
	Info     struct{}          `name:"更新问卷" desc:"更新问卷"`
	Request  UpdateApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response UpdateApiResponse // API响应数据 (Body中的Data部分)
}

type UpdateApiRequest struct {
	Body struct {
		ID     int64               `json:"id" binding:"required,gte=1" desc:"问卷ID"`
		Schema schema.SurveySchema `json:"schema" binding:"required" desc:"问卷结构"`
	}
}

type UpdateApiResponse struct{}

// Run Api业务逻辑执行点
func (u *UpdateApi) Run(ctx *gin.Context) kit.Code {
	req := u.Request.Body

	// 获取登录管理员信息
	admin, err := jwt.GetIdentity[comm.AdminIdentity](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	// 问卷结构校验
	if err := req.Schema.NormalizeAndVerify(); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("问卷结构校验失败")
		return comm.CodeParameterInvalid
	}

	// 查询旧问卷
	oldSurvey, err := repo.NewSurveyRepo().FindByID(ctx, req.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询旧问卷失败")
		return comm.CodeDatabaseError
	}
	if oldSurvey == nil {
		return comm.CodeDataNotFound
	}

	// 校验权限
	if oldSurvey.AdminID != admin.ID && admin.Type != comm.AdminTypeSuper {
		return comm.CodePermissionDenied
	}

	// 旧问卷结构反序列化
	var oldSchema schema.SurveySchema
	if err := sonic.UnmarshalString(oldSurvey.Schema, &oldSchema); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("旧问卷结构反序列化失败")
		return comm.CodeDataParseError
	}

	// 统计数据变更处理
	newStatsList := make([]*model.Stats, 0)
	oldItemMap := lo.KeyBy(oldSchema.QuestionConf.Items, func(item schema.QuestionItem) string {
		return item.ID
	})
	for _, newItem := range req.Schema.QuestionConf.Items {
		if oldItem, exists := oldItemMap[newItem.ID]; exists {
			// 大类题型是否变更
			if newItem.GetCategory() != oldItem.GetCategory() {
				nlog.Pick().WithContext(ctx).Warnf("题目类型不兼容: %s (%s -> %s)", newItem.ID, oldItem.Type, newItem.Type)
				return comm.CodeParameterInvalid
			}

			// 选项类题型 检查新增选项
			if newItem.IsOptionType() {
				oldOptionMap := lo.KeyBy(oldItem.Options, func(opt schema.Option) string {
					return opt.ID
				})
				for _, newOpt := range newItem.Options {
					if _, exists := oldOptionMap[newOpt.ID]; !exists {
						newStatsList = append(newStatsList, &model.Stats{
							SurveyID:   oldSurvey.ID,
							QuestionID: newItem.ID,
							OptionID:   newOpt.ID,
						})
					}
				}
			}
		} else if newItem.IsOptionType() {
			// 新增选项类题型
			for _, newOpt := range newItem.Options {
				newStatsList = append(newStatsList, &model.Stats{
					SurveyID:   oldSurvey.ID,
					QuestionID: newItem.ID,
					OptionID:   newOpt.ID,
				})
			}
		}
	}

	// 问卷结构序列化
	schemaStr, err := sonic.MarshalString(req.Schema)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("问卷结构序列化失败")
		return comm.CodeDataParseError
	}

	// 事务 更新问卷 -> 创建新增统计数据
	err = repo.Transaction(func(tx *query.Query) error {
		// 更新问卷
		if _, err := repo.NewSurveyRepo(tx).UpdateSchema(ctx, oldSurvey.ID, req.Schema.BannerConf.TitleConf.MainTitle, schemaStr); err != nil {
			return err
		}

		// 创建新增统计数据
		if len(newStatsList) > 0 {
			if err := repo.NewStatsRepo(tx).BatchCreate(ctx, newStatsList); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("更新问卷失败")
		return comm.CodeDatabaseError
	}

	// 删除问卷缓存
	if err := cache.NewSurveyCache().Del(ctx, oldSurvey.Path); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("删除问卷缓存失败")
	}

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (u *UpdateApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&u.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfUpdate API执行入口
func hfUpdate(ctx *gin.Context) {
	api := &UpdateApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, api.Response)
		} else {
			reply.Fail(ctx, code)
		}
	}
}
