package result

import (
	"fmt"
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
	"app/dao/repo"
	"app/schema"
)

// StatsHandler API router注册点
func StatsHandler() gin.HandlerFunc {
	api := StatsApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfStats).Pointer()).Name()] = api
	return hfStats
}

type StatsApi struct {
	Info     struct{}         `name:"获取答卷统计数据" desc:"获取答卷统计数据"`
	Request  StatsApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response StatsApiResponse // API响应数据 (Body中的Data部分)
}

type StatsApiRequest struct {
	Query struct {
		SurveyID int64 `form:"survey_id" binding:"required,gte=1" desc:"问卷ID"`
	}
}

type StatsApiResponse struct {
	List        []StatsItem `json:"list" desc:"统计数据列表"`
	SubmitCount int64       `json:"submit_count" desc:"提交总数"`
}

type StatsItem struct {
	ID      string            `json:"id" desc:"题目ID"`
	Title   string            `json:"title" desc:"题目标题"`
	Type    comm.QuestionType `json:"type" desc:"题型"`
	Options []Option          `json:"options" desc:"选项统计数据"`
}

type Option struct {
	ID    string `json:"id" desc:"选项ID"`
	Text  string `json:"text" desc:"选项文本"`
	Count int32  `json:"count" desc:"数量"`
}

// Run Api业务逻辑执行点
func (s *StatsApi) Run(ctx *gin.Context) kit.Code {
	req := s.Request.Query

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

	// 查询统计数据列表
	statsList, err := repo.NewStatsRepo().FindListBySurveyID(ctx, survey.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询统计数据列表失败")
		return comm.CodeDatabaseError
	}

	// 查询提交总数
	totalCount, err := repo.NewResultRepo().CountBySurveyID(ctx, survey.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询提交总数失败")
		return comm.CodeDatabaseError
	}
	s.Response.SubmitCount = totalCount

	// 构建统计数据映射 map[QuestionID][OptionID]Count
	statsMap := make(map[string]map[string]int32)
	for _, st := range statsList {
		if _, ok := statsMap[st.QuestionID]; !ok {
			statsMap[st.QuestionID] = make(map[string]int32)
		}
		statsMap[st.QuestionID][st.OptionID] = st.Count
	}

	// 筛选选项类题目
	optionQuestions := lo.Filter(surveySchema.QuestionConf.Items, func(item schema.QuestionItem, _ int) bool {
		return item.IsOptionType()
	})

	// 构建响应数据
	s.Response.List = lo.Map(optionQuestions, func(item schema.QuestionItem, _ int) StatsItem {
		options := make([]Option, 0, len(item.Options))

		// 处理现有选项
		processedOpts := make(map[string]bool)
		for _, opt := range item.Options {
			count := int32(0)
			if optCountMap, ok := statsMap[item.ID]; ok {
				count = optCountMap[opt.ID]
			}
			options = append(options, Option{
				ID:    opt.ID,
				Text:  opt.Text,
				Count: count,
			})
			processedOpts[opt.ID] = true
		}

		// 处理已删除选项
		if optCountMap, ok := statsMap[item.ID]; ok {
			delOptIDs := lo.Filter(lo.Keys(optCountMap), func(optID string, _ int) bool {
				return !processedOpts[optID]
			})
			delOpts := lo.Map(delOptIDs, func(optID string, i int) Option {
				return Option{
					ID:    optID,
					Text:  fmt.Sprintf("已删除选项(%s)", optID),
					Count: optCountMap[optID],
				}
			})
			options = append(options, delOpts...)
		}

		return StatsItem{
			ID:      item.ID,
			Title:   item.Title,
			Type:    item.Type,
			Options: options,
		}
	})

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (s *StatsApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindQuery(&s.Request.Query)
	if err != nil {
		return err
	}
	return err
}

// hfStats API执行入口
func hfStats(ctx *gin.Context) {
	api := &StatsApi{}
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
