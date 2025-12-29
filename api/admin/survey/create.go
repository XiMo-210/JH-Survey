package survey

import (
	"reflect"
	"runtime"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/model"
	"app/dao/query"
	"app/dao/repo"
	"app/schema"
)

// CreateHandler API router注册点
func CreateHandler() gin.HandlerFunc {
	api := CreateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfCreate).Pointer()).Name()] = api
	return hfCreate
}

type CreateApi struct {
	Info     struct{}          `name:"创建问卷" desc:"创建问卷"`
	Request  CreateApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response CreateApiResponse // API响应数据 (Body中的Data部分)
}

type CreateApiRequest struct {
	Body struct {
		Type   comm.SurveyType     `json:"type" binding:"required,oneof=1 2" desc:"问卷类型 1-问卷 2-投票"`
		Schema schema.SurveySchema `json:"schema" binding:"required" desc:"问卷结构"`
	}
}

type CreateApiResponse struct{}

// Run Api业务逻辑执行点
func (c *CreateApi) Run(ctx *gin.Context) kit.Code {
	req := c.Request.Body

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

	// 问卷结构序列化
	schemaStr, err := sonic.MarshalString(req.Schema)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("问卷结构序列化失败")
		return comm.CodeDataParseError
	}

	// 事务 创建问卷 -> 初始化统计数据
	survey := &model.Survey{
		AdminID: admin.ID,
		Title:   req.Schema.BannerConf.TitleConf.MainTitle,
		Type:    int8(req.Type),
		Path:    uuid.NewString(),
		Schema:  schemaStr,
		Status:  int8(comm.SurveyStatusUnpublished),
	}
	if err := repo.Transaction(func(tx *query.Query) error {
		// 创建问卷
		if err := repo.NewSurveyRepo(tx).Create(ctx, survey); err != nil {
			return err
		}

		// 初始化统计数据
		statsList := lo.FlatMap(req.Schema.QuestionConf.Items, func(item schema.QuestionItem, _ int) []*model.Stats {
			if !item.IsOptionType() {
				return nil
			}
			return lo.Map(item.Options, func(option schema.Option, _ int) *model.Stats {
				return &model.Stats{
					SurveyID:   survey.ID,
					QuestionID: item.ID,
					OptionID:   option.ID,
				}
			})
		})
		if len(statsList) > 0 {
			if err := repo.NewStatsRepo(tx).BatchCreate(ctx, statsList); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("创建问卷失败")
		return comm.CodeDatabaseError
	}

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (c *CreateApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&c.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfCreate API执行入口
func hfCreate(ctx *gin.Context) {
	api := &CreateApi{}
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
