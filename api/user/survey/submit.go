package survey

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

// SubmitHandler API router注册点
func SubmitHandler() gin.HandlerFunc {
	api := SubmitApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSubmit).Pointer()).Name()] = api
	return hfSubmit
}

type SubmitApi struct {
	Info     struct{}          `name:"提交问卷" desc:"提交问卷"`
	Request  SubmitApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response SubmitApiResponse // API响应数据 (Body中的Data部分)
}

type SubmitApiRequest struct {
	Body struct {
		ID     int64        `json:"id" binding:"required,gte=1" desc:"问卷ID"`
		Result []ResultItem `json:"result" binding:"required,min=1" desc:"答卷结果"`
	}
}

type ResultItem struct {
	QuestionID string `json:"question_id" binding:"required" desc:"题目ID"`
	Answer     string `json:"answer" binding:"required" desc:"回答"`
}

type SubmitApiResponse struct{}

// Run Api业务逻辑执行点
func (s *SubmitApi) Run(ctx *gin.Context) kit.Code {
	// TODO: 在此处编写接口业务逻辑
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (s *SubmitApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&s.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfSubmit API执行入口
func hfSubmit(ctx *gin.Context) {
	api := &SubmitApi{}
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
