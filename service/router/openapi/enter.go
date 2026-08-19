package openapi

import (
	"sun-panel/api/api_v1"
	"sun-panel/api/api_v1/middleware"

	"github.com/gin-gonic/gin"
)

func Init(rootRouter, apiRouter *gin.RouterGroup, cors gin.HandlerFunc) {
	openAPI := api_v1.ApiGroupApp.ApiOpenAPI.OpenAPI

	private := apiRouter.Group("/openApi", middleware.LoginInterceptor)
	{
		private.POST("/getConfig", openAPI.GetConfig)
		private.POST("/enable", openAPI.Enable)
		private.POST("/disable", openAPI.Disable)
		private.POST("/refreshToken", openAPI.RefreshToken)
	}

	external := rootRouter.Group("/openapi/v1")
	external.Use(cors, middleware.OpenAPIInterceptor)
	{
		external.POST("/version", openAPI.Version)
		external.POST("/itemGroup/getList", openAPI.GetItemGroupList)
		external.POST("/item/create", openAPI.CreateItem)
	}
}
