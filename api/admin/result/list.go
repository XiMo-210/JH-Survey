package result

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/model"
	"app/dao/repo"
	"app/schema"
)

// ListHandler API router注册点
func ListHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{}        `name:"获取答卷列表" desc:"获取答卷列表"`
	Request  ListApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response ListApiResponse // API响应数据 (Body中的Data部分)
}

type ListApiRequest struct {
	Query struct {
		SurveyID int64 `form:"survey_id" binding:"required,gte=1" desc:"问卷ID"`
		Page     int   `form:"page" binding:"required,gte=1" desc:"页码"`
		PageSize int   `form:"page_size" binding:"required,gte=1,lte=100" desc:"每页数量"`
	}
}

type ListApiResponse struct {
	Page     int                 `json:"page" desc:"页码"`
	PageSize int                 `json:"page_size" desc:"每页数量"`
	ListHead []QuestionItem      `json:"list_head" desc:"列表头"`
	ListBody [][]comm.ResultItem `json:"list_body" desc:"列表内容"`
	Total    int64               `json:"total" desc:"总数量"`
}

type QuestionItem struct {
	ID        string  `json:"id" desc:"题目ID"`
	Title     string  `json:"title" desc:"题目标题"`
	Type      string  `json:"type" desc:"题型"`
	OthersKey []Other `json:"others_key" desc:"自定义输入内容"`
}

type Other struct {
	Key    string `json:"key" desc:"自定义输入内容ID"`
	Option string `json:"option" desc:"自定义输入内容选项文本"`
}

// Run Api业务逻辑执行点
func (l *ListApi) Run(ctx *gin.Context) kit.Code {
	req := l.Request.Query
	l.Response.Page = req.Page
	l.Response.PageSize = req.PageSize

	// 获取登录管理员信息
	admin, err := jwt.GetIdentity[comm.AdminIdentity](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	// 查询问卷
	survey, err := repo.NewSurveyRepo().FindByID(ctx, req.SurveyID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
		return comm.CodeDatabaseError
	}
	if survey == nil {
		return comm.CodeDataNotFound
	}

	// 校验权限
	if survey.AdminID != admin.ID && admin.Type != comm.AdminTypeSuper {
		return comm.CodePermissionDenied
	}

	// 问卷结构反序列化
	var surveySchema schema.SurveySchema
	if err := sonic.UnmarshalString(survey.Schema, &surveySchema); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("问卷结构反序列化失败")
		return comm.CodeDataParseError
	}

	// 构建列表头
	l.Response.ListHead = lo.Map(surveySchema.QuestionConf.Items, func(item schema.QuestionItem, _ int) QuestionItem {
		// 自定义输入内容选项
		othersKey := lo.FilterMap(item.Options, func(opt schema.Option, _ int) (Other, bool) {
			return Other{
				Key:    opt.OthersKey,
				Option: opt.Text,
			}, opt.Others
		})
		return QuestionItem{
			ID:        item.ID,
			Title:     item.Title,
			Type:      string(item.Type),
			OthersKey: othersKey,
		}
	})

	// 查询答卷列表
	list, total, err := repo.NewResultRepo().FindPage(ctx, req.SurveyID, req.Page, req.PageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询答卷列表失败")
		return comm.CodeDatabaseError
	}
	l.Response.Total = total

	// 筛选选项类题目
	optionQuestions := lo.Filter(surveySchema.QuestionConf.Items, func(item schema.QuestionItem, _ int) bool {
		return item.IsOptionType()
	})

	// 构建题目选项映射 map[QuestionID]map[OptionID]OptionText
	questionOptionMap := lo.SliceToMap(optionQuestions, func(item schema.QuestionItem) (string, map[string]string) {
		return item.ID, lo.SliceToMap(item.Options, func(o schema.Option) (string, string) {
			return o.ID, o.Text
		})
	})

	// 构建响应数据
	l.Response.ListBody = lo.Map(list, func(res *model.Result, _ int) []comm.ResultItem {
		// 答卷数据反序列化
		var resultItems []comm.ResultItem
		if err := sonic.UnmarshalString(res.Data, &resultItems); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Errorf("答卷数据解析失败 ID:%d", res.ID)
			return nil
		}

		// 构建题目答案映射 map[QuestionID]Answer
		answerMap := lo.SliceToMap(resultItems, func(item comm.ResultItem) (string, string) {
			return item.QuestionID, item.Answer
		})

		// 构建响应行数据
		row := lo.FlatMap(surveySchema.QuestionConf.Items, func(item schema.QuestionItem, _ int) []comm.ResultItem {
			val := answerMap[item.ID]
			if item.IsOptionType() && val != "" {
				// 选项类题目将选项ID转换为文本
				if optMap, ok := questionOptionMap[item.ID]; ok {
					selectedIDs := strings.Split(val, ",")
					selectedTexts := lo.FilterMap(selectedIDs, func(id string, _ int) (string, bool) {
						if text, ok := optMap[id]; ok {
							return text, true
						}
						return "", false
					})
					val = strings.Join(selectedTexts, ",")
				}
			}

			items := []comm.ResultItem{{
				QuestionID: item.ID,
				Answer:     val,
			}}

			// 自定义输入内容选项
			if item.IsOptionType() {
				others := lo.FilterMap(item.Options, func(opt schema.Option, _ int) (comm.ResultItem, bool) {
					key := opt.OthersKey
					return comm.ResultItem{
						QuestionID: key,
						Answer:     answerMap[key],
					}, opt.Others
				})
				items = append(items, others...)
			}

			return items
		})

		return row
	})

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (l *ListApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindQuery(&l.Request.Query)
	if err != nil {
		return err
	}
	return err
}

// hfList API执行入口
func hfList(ctx *gin.Context) {
	api := &ListApi{}
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
