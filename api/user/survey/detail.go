package survey

import (
	"errors"
	"reflect"
	"runtime"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/sync/singleflight"

	"app/comm"
	"app/dao/cache"
	"app/dao/model"
	"app/dao/repo"
	"app/schema"
)

var sf singleflight.Group

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
		Path string `form:"path" binding:"required,max=64" desc:"访问路径"`
	}
}

type DetailApiResponse struct {
	ID     int64               `json:"id" desc:"问卷ID"`
	Type   comm.SurveyType     `json:"type" desc:"问卷类型"`
	Schema schema.SurveySchema `json:"schema" desc:"问卷结构"`
	Stats  []StatsItem         `json:"stats" desc:"选项统计数据"`
}

type StatsItem struct {
	ID      string   `json:"id" desc:"题目ID"`
	Options []Option `json:"options" desc:"统计数据"`
}

type Option struct {
	ID    string `json:"id" desc:"选项ID"`
	Count int32  `json:"count" desc:"数量"`
	Rank  int32  `json:"rank" desc:"排名"`
}

// Run Api业务逻辑执行点
func (d *DetailApi) Run(ctx *gin.Context) kit.Code {
	req := d.Request.Query

	// 查询问卷缓存
	survey, err := cache.NewSurveyCache().Get(ctx, req.Path)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷缓存失败")
	}
	// 缓存未命中 回源数据库
	if survey == nil {
		// singleflight 防止缓存穿透
		val, err, _ := sf.Do(req.Path, func() (any, error) {
			// 数据库查询问卷
			record, err := repo.NewSurveyRepo().FindByPath(ctx, req.Path)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
				return nil, err
			}
			if record == nil {
				return nil, kit.ErrNotFound
			}

			// 设置问卷缓存
			if err := cache.NewSurveyCache().Set(ctx, req.Path, record); err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("设置问卷缓存失败")
			}

			return record, nil
		})
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
			if errors.Is(err, kit.ErrNotFound) {
				return comm.CodeDataNotFound
			}
			return comm.CodeDatabaseError
		}

		// 断言类型
		var ok bool
		survey, ok = val.(*model.Survey)
		if !ok {
			nlog.Pick().WithContext(ctx).Error("singleflight 返回值类型断言失败")
			return comm.CodeUnknownError
		}
	}

	// 问卷结构反序列化
	var s schema.SurveySchema
	if err := sonic.UnmarshalString(survey.Schema, &s); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("问卷结构反序列化失败")
		return comm.CodeDataParseError
	}

	// TODO: 查询统计数据

	// 构建响应数据
	d.Response = DetailApiResponse{
		ID:     survey.ID,
		Type:   comm.SurveyType(survey.Type),
		Schema: s,
		Stats:  []StatsItem{},
	}

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
