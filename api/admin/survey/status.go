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

// StatusHandler API router注册点
func StatusHandler() gin.HandlerFunc {
	api := StatusApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfStatus).Pointer()).Name()] = api
	return hfStatus
}

type StatusApi struct {
	Info     struct{}          `name:"修改问卷状态" desc:"修改问卷状态"`
	Request  StatusApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response StatusApiResponse // API响应数据 (Body中的Data部分)
}

type StatusApiRequest struct {
	Body struct {
		ID     int64             `json:"id" binding:"required,gte=1" desc:"问卷ID"`
		Status comm.SurveyStatus `json:"status" binding:"required,oneof=1 2" desc:"修改状态 1-未发布 2-已发布"`
	}
}

type StatusApiResponse struct{}

// Run Api业务逻辑执行点
func (s *StatusApi) Run(ctx *gin.Context) kit.Code {
	// TODO: 在此处编写接口业务逻辑
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (s *StatusApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&s.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfStatus API执行入口
func hfStatus(ctx *gin.Context) {
	api := &StatusApi{}
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
