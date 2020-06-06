package main

import (
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go-ws/config"
	"go-ws/middlewares"
	"go-ws/router"
	"go-ws/services/wsservice"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	config.Init("")
	routers := gin.Default()
	pprof.Register(routers)

	// 全局中间件
	routers.Use(middlewares.Recovery())
	routers.Use(middlewares.RequestLogger())
	routers.Use(middlewares.ResponseHandler())
	routers.Use(middlewares.Cors())

	routers.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code": http.StatusNotFound,
			"msg":  "Api route not found",
		})
	})

	go wsservice.ManagerWsUserConnInfos()

	router.Router(routers)
	srv := &http.Server{
		Addr:    config.Settings.App.Bind,
		Handler: routers,
	}

	// gracehttp可平滑重启
	if err := gracehttp.Serve(srv); err != nil {
		logger.Logger.Info("Start Server failed", zap.Error(err))
		return
	}

	defer func(srv *http.Server) {
		err := recover()
		if err != nil {
			err := fmt.Errorf("panic %s", err)
			logger.Logger.Error("Server Shutdown:", zap.Error(err))
			return
		}
		logger.Logger.Error("Server Shutdown Success")
	}(srv)

	logger.Logger.Info("Start Server Success")

}
