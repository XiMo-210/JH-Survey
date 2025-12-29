package survey

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
	"app/dao/repo"
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
	req := s.Request.Body

	// 获取登录管理员信息
	admin, err := jwt.GetIdentity[comm.AdminIdentity](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	// 查询问卷
	survey, err := repo.NewSurveyRepo().FindByID(ctx, req.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
		return comm.CodeDatabaseError
	}
	if survey == nil {
		return comm.CodeDataNotFound
	}

	// 权限校验
	if survey.AdminID != admin.ID && admin.Type != comm.AdminTypeSuper {
		return comm.CodePermissionDenied
	}

	// 更新问卷状态
	if _, err := repo.NewSurveyRepo().UpdateStatus(ctx, req.ID, req.Status); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("更新问卷状态失败")
		return comm.CodeDatabaseError
	}

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
