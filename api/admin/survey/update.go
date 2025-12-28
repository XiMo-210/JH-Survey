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
	// TODO: 在此处编写接口业务逻辑
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
