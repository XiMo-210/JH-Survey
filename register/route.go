package register

import (
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/config"
	"github.com/zjutjh/mygo/middleware/cors"
	"github.com/zjutjh/mygo/swagger"

	"app/api"
	adminauth "app/api/admin/auth"
	adminresult "app/api/admin/result"
	adminsurvey "app/api/admin/survey"
	userauth "app/api/user/auth"
	usersurvey "app/api/user/survey"
)

func Route(router *gin.Engine) {
	router.Use(cors.Pick())

	r := router.Group(routePrefix())
	{
		routeBase(r, router)

		// 注册业务逻辑接口
		adminGroup := r.Group("/admin")
		{
			authGroup := adminGroup.Group("/auth")
			{
				authGroup.POST("/create", adminauth.CreateHandler()) // 创建管理员
				authGroup.POST("/login", adminauth.LoginHandler())   // 管理员登录
			}
			surveyGroup := adminGroup.Group("/survey")
			{
				surveyGroup.GET("/detail", adminsurvey.DetailHandler())  // 获取问卷详情
				surveyGroup.GET("/list", adminsurvey.ListHandler())      // 获取问卷列表
				surveyGroup.POST("/create", adminsurvey.CreateHandler()) // 创建问卷
				surveyGroup.POST("/update", adminsurvey.UpdateHandler()) // 更新问卷
				surveyGroup.POST("/status", adminsurvey.StatusHandler()) // 修改问卷状态
				surveyGroup.POST("/delete", adminsurvey.DeleteHandler()) // 删除问卷
			}
			resultGroup := adminGroup.Group("/result")
			{
				resultGroup.GET("/stats", adminresult.StatsHandler()) // 获取答卷统计数据
				resultGroup.GET("/list", adminresult.ListHandler())   // 获取答卷列表
			}
		}

		userGroup := r.Group("/user")
		{
			authGroup := userGroup.Group("/auth")
			{
				authGroup.POST("/login", userauth.LoginHandler()) // 用户登录
			}
			surveyGroup := userGroup.Group("/survey")
			{
				surveyGroup.GET("/detail", usersurvey.DetailHandler())  // 获取问卷详情
				surveyGroup.POST("/submit", usersurvey.SubmitHandler()) // 提交问卷
			}
		}
	}
}

func routePrefix() string {
	return "/api"
}

func routeBase(r *gin.RouterGroup, router *gin.Engine) {
	// OpenAPI/Swagger 文档生成
	if slices.Contains([]string{config.AppEnvDev, config.AppEnvTest}, config.AppEnv()) {
		r.GET("/swagger.json", swagger.DocumentHandler(router))
	}

	// 健康检查
	r.GET("/health", api.HealthHandler())
}
