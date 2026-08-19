package middleware

import (
	"strings"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/global"
	"sun-panel/models"

	"github.com/gin-gonic/gin"
)

// OpenAPIInterceptor resolves a browser-extension token to its owning user.
func OpenAPIInterceptor(c *gin.Context) {
	token := strings.TrimSpace(c.GetHeader("token"))
	if token == "" {
		apiReturn.ErrorByCode(c, 1000)
		c.Abort()
		return
	}

	config := models.OpenAPIConfig{}
	err := global.Db.Preload("User").
		Where("token = ? AND enabled = ?", token, true).
		First(&config).Error
	if err != nil || config.User.ID == 0 || config.User.Status != 1 {
		apiReturn.ErrorByCode(c, 1000)
		c.Abort()
		return
	}

	c.Set("userInfo", config.User)
}
