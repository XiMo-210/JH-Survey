package survey

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/model"
	"app/dao/repo"
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
	Admin     string            `json:"admin" desc:"所属管理员"`
	Title     string            `json:"title" desc:"问卷标题"`
	Type      comm.SurveyType   `json:"type" desc:"问卷类型"`
	Path      string            `json:"path" desc:"访问路径"`
	Status    comm.SurveyStatus `json:"status" desc:"状态 1-未发布 2-已发布"`
	CreatedAt string            `json:"created_at" desc:"创建时间"`
	UpdatedAt string            `json:"updated_at" desc:"更新时间"`
}

// Run Api业务逻辑执行点
func (l *ListApi) Run(ctx *gin.Context) kit.Code {
	req := l.Request.Query
	l.Response.Page = req.Page
	l.Response.PageSize = req.PageSize

	// 获取登录管理员信息
	admin, err := jwt.GetIdentity[comm.AdminIdentity](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	// 查询条件
	adminID := int64(0)
	if admin.Type != comm.AdminTypeSuper {
		adminID = admin.ID
	}

	// 查询问卷列表
	list, total, err := repo.NewSurveyRepo().FindPage(ctx, req.Page, req.PageSize, adminID, req.Type, req.Status, req.Keyword)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷列表失败")
		return comm.CodeDatabaseError
	}
	l.Response.Total = total

	// 构建管理员映射
	adminMap := map[int64]string{
		admin.ID: admin.Username,
	}
	if admin.Type == comm.AdminTypeSuper && len(list) > 0 {
		adminIDs := lo.Uniq(lo.Map(list, func(item *model.Survey, _ int) int64 {
			return item.AdminID
		}))
		// 查询管理员列表
		admins, err := repo.NewAdminRepo().FindListByIDs(ctx, adminIDs)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Error("查询管理员列表失败")
			return comm.CodeDatabaseError
		}
		adminMap = lo.SliceToMap(admins, func(item *model.Admin) (int64, string) {
			return item.ID, item.Username
		})
	}

	// 构建响应数据
	l.Response.List = lo.Map(list, func(item *model.Survey, _ int) SurveyItem {
		return SurveyItem{
			ID:        item.ID,
			Admin:     adminMap[item.AdminID],
			Title:     item.Title,
			Type:      comm.SurveyType(item.Type),
			Path:      item.Path,
			Status:    comm.SurveyStatus(item.Status),
			CreatedAt: item.CreatedAt.Format(time.DateTime),
			UpdatedAt: item.UpdatedAt.Format(time.DateTime),
		}
	})

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
