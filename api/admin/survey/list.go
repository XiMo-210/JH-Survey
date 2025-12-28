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

// ListHandler API router注册点
func ListHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{}        `name:"获取问卷列表" desc:"获取问卷列表"`
	Request  ListApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response ListApiResponse // API响应数据 (Body中的Data部分)
}

type ListApiRequest struct {
	Query struct {
		Page     int               `form:"page" binding:"required,gte=1" desc:"页码"`
		PageSize int               `form:"page_size" binding:"required,gte=1,lte=100" desc:"每页数量"`
		Type     comm.SurveyType   `form:"type" binding:"omitempty,oneof=1 2" desc:"问卷类型"`
		Status   comm.SurveyStatus `form:"status" binding:"omitempty,oneof=1 2" desc:"状态 1-未发布 2-已发布"`
		Keyword  string            `form:"keyword" binding:"omitempty,max=64" desc:"搜索关键词"`
	}
}

type ListApiResponse struct {
	Page     int          `json:"page" desc:"页码"`
	PageSize int          `json:"page_size" desc:"每页数量"`
	List     []SurveyItem `json:"list" desc:"问卷列表"`
	Total    int64        `json:"total" desc:"总数量"`
}

type SurveyItem struct {
	ID        int64             `json:"id" desc:"问卷ID"`
	Title     string            `json:"title" desc:"问卷标题"`
	Type      comm.SurveyType   `json:"type" desc:"问卷类型"`
	Path      string            `json:"path" desc:"访问路径"`
	Status    comm.SurveyStatus `json:"status" desc:"状态 1-未发布 2-已发布"`
	CreatedAt int64             `json:"created_at" desc:"创建时间"`
	UpdatedAt int64             `json:"updated_at" desc:"更新时间"`
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
