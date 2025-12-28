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

// DeleteHandler API router注册点
func DeleteHandler() gin.HandlerFunc {
	api := DeleteApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDelete).Pointer()).Name()] = api
	return hfDelete
}

type DeleteApi struct {
	Info     struct{}          `name:"删除问卷" desc:"删除问卷"`
	Request  DeleteApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response DeleteApiResponse // API响应数据 (Body中的Data部分)
}

type DeleteApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required,gte=1" desc:"问卷ID"`
	}
}

type DeleteApiResponse struct{}

// Run Api业务逻辑执行点
func (d *DeleteApi) Run(ctx *gin.Context) kit.Code {
	// TODO: 在此处编写接口业务逻辑
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (d *DeleteApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&d.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfDelete API执行入口
func hfDelete(ctx *gin.Context) {
	api := &DeleteApi{}
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
