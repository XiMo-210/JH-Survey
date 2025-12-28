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
	ListBody []map[string]string `json:"list_body" desc:"列表内容"`
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
	// TODO: 在此处编写接口业务逻辑
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
