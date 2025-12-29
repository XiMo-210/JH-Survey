package auth

import (
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
)

// InfoHandler API router注册点
func InfoHandler() gin.HandlerFunc {
	api := InfoApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfInfo).Pointer()).Name()] = api
	return hfInfo
}

type InfoApi struct {
	Info     struct{}        `name:"获取管理员信息" desc:"获取管理员信息"`
	Request  InfoApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response InfoApiResponse // API响应数据 (Body中的Data部分)
}

type InfoApiRequest struct {
}

type InfoApiResponse struct {
	Username string         `json:"username" desc:"用户名"`
	Type     comm.AdminType `json:"type" desc:"用户类型"`
}

// Run Api业务逻辑执行点
func (i *InfoApi) Run(ctx *gin.Context) kit.Code {
	// 获取登录管理员信息
	admin, err := jwt.GetIdentity[comm.AdminIdentity](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	i.Response.Username = admin.Username
	i.Response.Type = admin.Type

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (i *InfoApi) Init(ctx *gin.Context) (err error) {
	return err
}

// hfInfo API执行入口
func hfInfo(ctx *gin.Context) {
	api := &InfoApi{}
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
