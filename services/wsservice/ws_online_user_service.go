package wsservice

import (
	"github.com/gomodule/redigo/redis"
	myredis "go-ws/databases/redis"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"strconv"
)

// 添加链接的用户ID
func AddOnlineUserId(userId int) (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := wsUserOnlineListKey + strconv.Itoa(userId)

	// set集合不会有相同的值
	_, err = rd.Do("sAdd", cacheKey, userId)
	if err != nil {
		logger.Logger.Warn("add websocket user id failed", zap.Int("user_id", userId), zap.Error(err))
		return
	}

	return
}

// 删除链接的用户ID
func DelOnlineUserId(userId int) (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := wsUserOnlineListKey + strconv.Itoa(userId)

	_, err = rd.Do("sRem", cacheKey, userId)
	if err != nil {
		logger.Logger.Warn("add websocket user id failed", zap.Int("user_id", userId), zap.Error(err))
		return
	}

	return
}


// 添加链接过的用户ID列表
func GetOnLineUserIdList() (userIdList []int, err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := wsUserOnlineListKey

	userIdList, err = redis.Ints(rd.Do("sMembers", cacheKey))
	if err != nil {
		logger.Logger.Warn("get websocket user list failed", zap.Error(err))
		return
	}
	return
}
