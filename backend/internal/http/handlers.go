package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter M1 仅暴露健康检查；业务 API 从 M2 开始挂载。
func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	r.GET("/healthz", Healthz)
	return r
}

func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
