package survey

import (
	"errors"
	"reflect"
	"runtime"
	"sort"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/sync/singleflight"

	"app/comm"
	"app/dao/cache"
	"app/dao/model"
	"app/dao/repo"
	"app/schema"
)

var sf singleflight.Group

// DetailHandler API router注册点
func DetailHandler() gin.HandlerFunc {
	api := DetailApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDetail).Pointer()).Name()] = api
	return hfDetail
}

type DetailApi struct {
	Info     struct{}          `name:"获取问卷详情" desc:"获取问卷详情"`
	Request  DetailApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response DetailApiResponse // API响应数据 (Body中的Data部分)
}

type DetailApiRequest struct {
	Query struct {
		Path string `form:"path" binding:"required,max=64" desc:"访问路径"`
	}
}

type DetailApiResponse struct {
	ID     int64               `json:"id" desc:"问卷ID"`
	Type   comm.SurveyType     `json:"type" desc:"问卷类型"`
	Schema schema.SurveySchema `json:"schema" desc:"问卷结构"`
	Stats  []StatsItem         `json:"stats" desc:"选项统计数据"`
}

type StatsItem struct {
	ID      string   `json:"id" desc:"题目ID"`
	Options []Option `json:"options" desc:"统计数据"`
}

type Option struct {
	ID    string `json:"id" desc:"选项ID"`
	Count int32  `json:"count" desc:"数量"`
	Rank  int32  `json:"rank" desc:"排名"`
}

// Run Api业务逻辑执行点
func (d *DetailApi) Run(ctx *gin.Context) kit.Code {
	req := d.Request.Query

	// 查询问卷缓存
	survey, err := cache.NewSurveyCache().Get(ctx, req.Path)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷缓存失败")
	}
	// 缓存未命中 回源数据库
	if survey == nil {
		// singleflight 防止缓存穿透
		val, err, _ := sf.Do(req.Path, func() (any, error) {
			// 数据库查询问卷
			record, err := repo.NewSurveyRepo().FindByPath(ctx, req.Path)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
				return nil, err
			}
			if record == nil {
				return nil, kit.ErrNotFound
			}

			// 设置问卷缓存
			if err := cache.NewSurveyCache().Set(ctx, req.Path, record); err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("设置问卷缓存失败")
			}

			return record, nil
		})
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
			if errors.Is(err, kit.ErrNotFound) {
				return comm.CodeDataNotFound
			}
			return comm.CodeDatabaseError
		}

		// 断言类型
		var ok bool
		survey, ok = val.(*model.Survey)
		if !ok {
			nlog.Pick().WithContext(ctx).Error("singleflight 返回值类型断言失败")
			return comm.CodeUnknownError
		}
	}

	// 问卷结构反序列化
	var surveySchema schema.SurveySchema
	if err := sonic.UnmarshalString(survey.Schema, &surveySchema); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("问卷结构反序列化失败")
		return comm.CodeDataParseError
	}

	// 筛选需要显示统计数据的投票类题目
	voteQuestions := lo.Filter(surveySchema.QuestionConf.Items, func(item schema.QuestionItem, _ int) bool {
		return item.IsVoteType() && item.ShowStats
	})

	stats := make([]StatsItem, 0)

	if len(voteQuestions) > 0 {
		// 检查是否提交后才能查看数据
		hasSubmitted := false
		needSubmit := lo.ContainsBy(voteQuestions, func(item schema.QuestionItem) bool {
			return item.ShowStatsAfterSubmit
		})
		if needSubmit {
			user, err := jwt.GetIdentity[comm.UserIdentity](ctx)
			if err == nil {
				count, err := repo.NewResultRepo().CountByUser(ctx, survey.ID, user.Username, nil)
				if err == nil && count > 0 {
					hasSubmitted = true
				}
			}
		}

		// 筛选实际需要展示统计数据的题目
		finalVoteQuestions := lo.Filter(voteQuestions, func(item schema.QuestionItem, _ int) bool {
			return !item.ShowStatsAfterSubmit || (item.ShowStatsAfterSubmit && hasSubmitted)
		})

		if len(finalVoteQuestions) > 0 {
			// 查询统计数据列表
			statsList, err := repo.NewStatsRepo().FindListBySurveyID(ctx, survey.ID)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("查询统计数据列表失败")
			}

			// 构建统计数据映射 map[QuestionID][OptionID]Count
			statsMap := make(map[string]map[string]int32)
			for _, st := range statsList {
				if _, ok := statsMap[st.QuestionID]; !ok {
					statsMap[st.QuestionID] = make(map[string]int32)
				}
				statsMap[st.QuestionID][st.OptionID] = st.Count
			}

			// 构建统计数据
			stats = lo.Map(finalVoteQuestions, func(item schema.QuestionItem, _ int) StatsItem {
				options := make([]Option, 0, len(item.Options))

				// 处理选项
				for _, opt := range item.Options {
					count := int32(0)
					if optCountMap, ok := statsMap[item.ID]; ok {
						count = optCountMap[opt.ID]
					}
					options = append(options, Option{
						ID:    opt.ID,
						Count: count,
					})
				}

				// 计算排名
				if item.ShowRank {
					allCounts := lo.Map(options, func(o Option, _ int) int32 {
						return o.Count
					})
					sort.Slice(allCounts, func(i, j int) bool {
						return allCounts[i] > allCounts[j]
					})
					rankMap := make(map[int32]int32)
					for i, c := range allCounts {
						if _, ok := rankMap[c]; !ok {
							rankMap[c] = int32(i + 1)
						}
					}
					for i := range options {
						options[i].Rank = rankMap[options[i].Count]
					}
				}

				return StatsItem{
					ID:      item.ID,
					Options: options,
				}
			})
		}
	}

	// 构建响应数据
	d.Response = DetailApiResponse{
		ID:     survey.ID,
		Type:   comm.SurveyType(survey.Type),
		Schema: surveySchema,
		Stats:  stats,
	}

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (d *DetailApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindQuery(&d.Request.Query)
	if err != nil {
		return err
	}
	return err
}

// hfDetail API执行入口
func hfDetail(ctx *gin.Context) {
	api := &DetailApi{}
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
