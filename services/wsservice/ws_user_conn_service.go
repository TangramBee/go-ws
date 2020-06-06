package wsservice

import (
	"github.com/gomodule/redigo/redis"
	myredis "go-ws/databases/redis"
	"go-ws/utils/http"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"net/url"
	"strconv"
	"time"
)

// 添加链接ID
func AddWsUserConnId(onlineUserId int, userConnId string) (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := wsUserConnectionListPreCacheKey + strconv.Itoa(onlineUserId)

	_, err = rd.Do("zAdd", cacheKey, time.Now().Unix(), userConnId)
	if err != nil {
		logger.Logger.Warn("add user connection id failed", zap.Int("user_id", onlineUserId), zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}

	logger.Logger.Info("add user connection id success", zap.Int("user_id", onlineUserId), zap.String("user_conn_id", userConnId))

	return
}

// 删除链接ID
func DelWsUserConnId(onlineUserId int, userConnId string) (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := wsUserConnectionListPreCacheKey + strconv.Itoa(onlineUserId)

	_, err = rd.Do("zRem", cacheKey, userConnId)
	if err != nil {
		logger.Logger.Warn("del user connection id failed", zap.Int("user_id", onlineUserId), zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}
	return
}

// （可多个设备登陆同一账号）获取用户链接ID列表，最多30个
func GetWsUserConnIdList(userId int) (userConnIdList []string, err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := wsUserConnectionListPreCacheKey + strconv.Itoa(userId)

	userConnIdList, err = redis.Strings(rd.Do("zRevRange", cacheKey, 0, 30))
	if err != nil {
		logger.Logger.Warn("get websocket client list failed", zap.Int("user_id", userId), zap.String("cacheKey", cacheKey), zap.Error(err))
		return
	}
	return
}

// 删除本机的用户链接
func DelLocalUserConn(userConnId string) (err error) {
	if w, ok := AllWsUserConnInfos[userConnId]; ok {
		err = w.wsConnection.Close()
		if err != nil {
			logger.Logger.Warn("del websocket user connection failed", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID), zap.Error(err))
			return
		}
		delete(AllWsUserConnInfos, userConnId)
	}

	return
}

// 删除其他服务器的用户链接
func DelOtherServerUserConn(node, userConnId string) (err error) {
	reqUrl := "http://" + node + "/ws/connection/close"
	var data = url.Values{}
	data.Add("cid", userConnId)
	err = http.Post(reqUrl, data)
	if err != nil {
		logger.Logger.Warn("delete websocket connection in other server failed", zap.String("node", node), zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}
	return
}


