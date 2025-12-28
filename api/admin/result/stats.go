package result

import (
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
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
	List []StatsItem `json:"list" desc:"统计数据列表"`
}

type StatsItem struct {
	ID    string        `json:"id" desc:"题目ID"`
	Title string        `json:"title" desc:"题目标题"`
	Type  string        `json:"type" desc:"题型"`
	Stats []OptionStats `json:"stats" desc:"选项统计数据"`
}

type OptionStats struct {
	Options     []Option `json:"options" desc:"选项统计数据"`
	SubmitCount int64    `json:"submit_count" desc:"提交总数"`
}

type Option struct {
	ID    string `json:"id" desc:"选项ID"`
	Text  string `json:"text" desc:"选项文本"`
	Count int32  `json:"count" desc:"数量"`
}

// Run Api业务逻辑执行点
func (s *StatsApi) Run(ctx *gin.Context) kit.Code {
	// TODO: 在此处编写接口业务逻辑
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
