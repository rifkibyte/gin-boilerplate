package router

import (
	"github.com/gin-gonic/gin"

	"github.com/rifkibyte/gin-boilerplate/internal/middleware"
	userhttp "github.com/rifkibyte/gin-boilerplate/internal/user/http"
)

// New builds the Gin engine with all routes registered.
func New(userHandler *userhttp.Handler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.AccessLog(), middleware.Recovery())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthCheck)
		v1.POST("/users", userHandler.Create)
	}

	return r
}

// healthCheck returns 200 when the server is alive.
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
