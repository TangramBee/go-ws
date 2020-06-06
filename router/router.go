package router

import (
	"github.com/gin-gonic/gin"
	"go-ws/handler"
)

func Router(router *gin.Engine) *gin.Engine {
	// im前端路由
	wsRouter := router.Group("/ws/")//.Use(middlewares.LoginAuth())
	{
		// 创建链接
		wsRouter.GET("connection/add", handler.WsConnectionHandler)

		// 关闭用户链接
		wsRouter.POST("connection/close", handler.CloseWsHandler)

		// 推送消息给某个用户链接，用于服务间推送
		wsRouter.POST("msg/push", handler.PushMsgToUserConn)

		// 接收信息保存到消息队列
		wsRouter.POST("msg/send", handler.SendMessageHandler)

	}

	return router
}

