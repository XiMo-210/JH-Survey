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
		ID int64 `form:"id" binding:"required,gte=1" desc:"问卷ID"`
	}
}

type DetailApiResponse struct {
	ID        int64               `json:"id" desc:"问卷ID"`
	Type      comm.SurveyType     `json:"type" desc:"问卷类型"`
	Path      string              `json:"path" desc:"访问路径"`
	Schema    schema.SurveySchema `json:"schema" desc:"问卷结构"`
	Status    comm.SurveyStatus   `json:"status" desc:"状态 1-未发布 2-已发布"`
	CreatedAt string              `json:"created_at" desc:"创建时间"`
	UpdatedAt string              `json:"updated_at" desc:"更新时间"`
}

// Run Api业务逻辑执行点
func (d *DetailApi) Run(ctx *gin.Context) kit.Code {
	// TODO: 在此处编写接口业务逻辑
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
