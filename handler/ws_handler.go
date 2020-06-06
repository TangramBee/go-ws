package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-ws/config"
	"go-ws/services/wsservice"
	"go-ws/utils"
	"go-ws/utils/errs"
	"go-ws/utils/logger"
	"go-ws/utils/ws"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 建立链接
func WsConnectionHandler(c *gin.Context) {
	uid, ok := strconv.Atoi(c.Query("uid"))
	if ok != nil || uid == 0  {
		c.Error(errs.ErrParam)
	}

	conn, err := ws.CreateWsConnection(c.Writer, c.Request)
	if err != nil {
		logger.Logger.Warn("create websocket connect failed", zap.Int("uid", uid), zap.Error(err))
		c.Error(err)
		return
	}

	ip, _ := utils.GetLocalIP()
	port := strings.Split(config.Settings.App.Bind, ":")[1]

	wsUserConn := wsservice.AddWsUserConnInfo(uid, ip + ":" + port, &conn)

	// 检测心跳
	go wsUserConn.HeartBeatCheck()
	// 循环推送消息
	go wsUserConn.PushLoop()
	// 循环接收消息，保存ACK
	go wsUserConn.ReceiveLoop()

}

// 关闭链接
func CloseWsHandler(c *gin.Context) {
	uid, ok := c.MustGet("uid").(int)
	if !ok || uid == 0  {
		c.Error(errs.ErrParam)
	}
	connId := c.PostForm("cid")
	if connId == "" {
		c.Error(errs.ErrWebSocketConnectionIDIsNil)
		return
	}

	wsConn, err := wsservice.GetWsUserConnInfo(connId)
	if err != nil {
		logger.Logger.Warn("websocket connection id is nil", zap.Int("uid", uid), zap.String("user_conn_id", connId), zap.Error(err))
		c.Error(err)
		return
	}

	if wsConn.Node == c.Request.Host {
		err = wsservice.DelLocalUserConn(connId)
	} else {
		err = wsservice.DelOtherServerUserConn(wsConn.Node, connId)
	}
	if err != nil {
		logger.Logger.Warn("delete websocket connection failed", zap.Int("uid", uid), zap.String("user_conn_id", connId), zap.Error(err))
		c.Error(err)
		return
	}

	wsConn.Closed = true
	wsConn.DisConnectTime = time.Now().Unix()
	// 更新链接状态
	err = wsConn.UpdateUserInfo()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// 关闭某个用户的所有链接
func CloseAllConnHandler(c *gin.Context) {
	uid, ok := c.MustGet("uid").(int)
	if !ok || uid == 0  {
		c.Error(errs.ErrParam)
	}

	var userConnList []wsservice.WsUserConnInfo
	userConnList, err := wsservice.GetAllUserInfoList(uid)
	if err != nil {
		logger.Logger.Warn("get user conn list failed", zap.Int("uid", uid), zap.Error(err))
		return
	}

	for _, userConn := range userConnList {
		if !userConn.Closed {
			if userConn.Node == c.Request.Host {
				err = wsservice.DelLocalUserConn(userConn.ID)
			} else {
				err = wsservice.DelOtherServerUserConn(userConn.Node, userConn.ID)
			}

			if err != nil {
				logger.Logger.Warn("delete websocket connection failed", zap.Int("uid", uid), zap.String("user_conn_id", userConn.ID), zap.Error(err))
				continue
			}

		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

type PushMsgReq struct {
	Uid int `json:"uid" form:"uid" binding:"required"`
	Content string `json:"content" form:"content" binding:"required"`
	Retries int `json:"retries" form:"retries" binding:"required"` // 重试次数
}

// 接收信息保存到消息队列
func SendMessageHandler(c *gin.Context) {
	var msgReq PushMsgReq
	if err := c.ShouldBind(&msgReq); err != nil {
		c.Error(err)
		return
	}

	msg := wsservice.Msg{
		ID:      fmt.Sprintf("%d-%s", time.Now().Unix(), uuid.New().String()),
		UID:     msgReq.Uid,
		Content: msgReq.Content,
		Retries: msgReq.Retries,
	}

	err := msg.PushWsMsgToQueue()
	if err != nil {
		c.Error(errs.ErrPushMsgToQueueFailed)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})

}

type RecMsgReq struct {
	ConnId string `json:"conn_id" form:"conn_id" binding:"required"`
	Content string `json:"content" form:"content" binding:"required"`
}

// 推送消息给某个用户链接，用于服务间推送
func PushMsgToUserConn(c *gin.Context) {
	var msgReq RecMsgReq
	if err := c.ShouldBind(&msgReq); err != nil {
		c.Error(err)
		return
	}

	var msg wsservice.Msg
	err := json.Unmarshal([]byte(msgReq.Content), &msg)
	if err != nil {
		c.Error(errs.ErrParam)
		return
	}

	err = msg.PushMsg(msgReq.ConnId)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}