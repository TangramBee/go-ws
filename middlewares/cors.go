package middlewares

import (
	"github.com/gin-gonic/gin"
)

// Cors across domain
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 可由nginx接入层来处理
		c.Header("Access-Control-Allow-Origin", "*") //测试使用
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Action, Module, X-PINGOTHER, Content-Type, Content-Disposition")
		c.Header("Access-Control-Allow-Credentials", "true")

		c.Next()
	}
}
