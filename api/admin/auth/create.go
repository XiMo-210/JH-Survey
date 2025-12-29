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
	"app/dao/model"
	"app/dao/repo"
)

// CreateHandler API router注册点
func CreateHandler() gin.HandlerFunc {
	api := CreateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfCreate).Pointer()).Name()] = api
	return hfCreate
}

type CreateApi struct {
	Info     struct{}          `name:"创建管理员" desc:"仅能创建普通管理员"`
	Request  CreateApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response CreateApiResponse // API响应数据 (Body中的Data部分)
}

type CreateApiRequest struct {
	Body struct {
		Username string `json:"username" binding:"required,max=16" desc:"用户名"`
		Password string `json:"password" binding:"required,max=32" desc:"密码"`
		Secret   string `json:"secret" binding:"max=64" desc:"密钥 超管登录可忽略"`
	}
}

type CreateApiResponse struct{}

// Run Api业务逻辑执行点
func (c *CreateApi) Run(ctx *gin.Context) kit.Code {
	req := c.Request.Body

	// 获取登录管理员信息
	admin, err := jwt.GetIdentity[comm.AdminIdentity](ctx)
	if err != nil || admin.Type != comm.AdminTypeSuper {
		// 非超管登录 校验密钥
		if req.Secret != comm.BizConf.AdminCreateSecret {
			nlog.Pick().WithContext(ctx).Warn("创建管理员密钥错误")
			return comm.CodePermissionDenied
		}
	}

	// 查询用户名是否存在
	record, err := repo.NewAdminRepo().FindByUsername(ctx, req.Username)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询管理员失败")
		return comm.CodeDatabaseError
	}
	if record != nil {
		return comm.CodeAdminAlreadyExist
	}

	// 密码加密
	password, err := comm.HashPassword(req.Password)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("密码加密失败")
		return comm.CodeUnknownError
	}

	// 创建管理员
	if err := repo.NewAdminRepo().Create(ctx, &model.Admin{
		Username: req.Username,
		Password: password,
		Type:     int8(comm.AdminTypeNormal),
	}); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("创建管理员失败")
		return comm.CodeDatabaseError
	}

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (c *CreateApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&c.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfCreate API执行入口
func hfCreate(ctx *gin.Context) {
	api := &CreateApi{}
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
