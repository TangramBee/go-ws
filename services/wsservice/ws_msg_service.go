package wsservice

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	myredis "go-ws/databases/redis"
	"go-ws/utils/http"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"net/url"
	"strconv"
	"time"
)

// 消息结构体
type Msg struct {
	ID string `json:"id"`
	UID int `json:"uid"`
	Content interface{} `json:"content"`
	Retries int `json:"retries"`
	ConnId  string `json:"conn_id"`
}

// 接收消息
type RecMsg struct {
	ID string `json:"id"` // 与Msg里的ID一致
	Content interface{} `json:"content"`
}

const (
	// 消息事件队列前缀
	msgQueuePreCacheKey = "ws_user_msg_queue:"
	// 消息事件推送失败的延迟队列
	msgDelayQueuePreCacheKey = "ws_user_msg_delay_queue:"
	// 消息收到ack后的消息ID
	msgAckPreCacheKey = "ws_user_msg_send_ack_list:"
	// 消息延迟重新推送间隔时间(s)
	msgDelayTimeOut = 3
)

// 消息事件写入队列
func (m *Msg) PushWsMsgToQueue() (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	var data []byte
	data, _ = json.Marshal(m)

	cacheKey := msgQueuePreCacheKey + strconv.Itoa(m.UID)
	_, err = rd.Do("lPush", cacheKey, string(data))
	if err != nil {
		logger.Logger.Warn("lPush websocket user msg to queue failed", zap.Any("msg", m), zap.String("cacheKey", cacheKey), zap.Error(err))
		return
	}

	logger.Logger.Info("lPush websocket user msg to queue success", zap.Any("msg", m), zap.String("cacheKey", cacheKey))

	return
}

// 获取消息事件内容
func PopWsMsgFromQueue(userId int) (msg Msg, err error) {
	rd := myredis.NewRedis("default_redis").Get()

	var data []byte
	cacheKey := msgQueuePreCacheKey + strconv.Itoa(userId)
	data, err = redis.Bytes(rd.Do("lPop", cacheKey))
	if err != nil {
		logger.Logger.Warn("lPop websocket user msg from queue failed", zap.String("cacheKey", cacheKey), zap.Error(err))
		return
	}

	err = json.Unmarshal(data, &msg)
	if err != nil {
		logger.Logger.Warn("lPop websocket user msg from queue json unmarshal failed", zap.String("cacheKey", cacheKey), zap.Int("user_id", userId), zap.ByteString("data", data), zap.Error(err))
		return
	}
	return
}

// 消息事件推送失败的延迟队列，增加排队时间
func (m *Msg) PushWsMsgToDelayQueue() (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	var data []byte
	data, _ = json.Marshal(m)

	cacheKey := msgDelayQueuePreCacheKey + strconv.Itoa(m.UID)
	_, err = rd.Do("zAdd", cacheKey, time.Now().Unix() + msgDelayTimeOut, string(data))
	if err != nil {
		logger.Logger.Warn(" websocket msg to delay queue failed", zap.Any("msg", m), zap.Error(err))
		return
	}

	logger.Logger.Info("zAdd websocket msg to delay queue success", zap.Any("msg", m))
	return
}

// 从推送失败的延时队列读取消息
func PopWsMsgFromDelayQueue(userId int) (msg Msg, err error) {
	rd := myredis.NewRedis("default_redis").Get()

	// 执行lua脚本，实现事务，使查删两个操作具有原子性
	luaScript := `
	local message = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'WITHSCORES', 'LIMIT', 0, 1)
	if #message > 0 then
		redis.Call('ZREM', KEYS[1], message[1]);
		return message[1];
	else
		return nil;
	end
	`

	script := redis.NewScript(1, luaScript)
	var cacheKey = msgDelayQueuePreCacheKey + strconv.Itoa(userId)

	var data []byte
	data, err = redis.Bytes(script.Do(rd, cacheKey, time.Now().Unix()))
	if err != nil {
		logger.Logger.Debug("zrangebyscore websocket msg from delay queue failed", zap.String("cacheKey", cacheKey), zap.Error(err))
		return
	}

	err = json.Unmarshal(data, &msg)
	if err != nil {
		logger.Logger.Warn("zrangebyscore websocket msg from delay queue json unmarshal failed", zap.String("cacheKey", cacheKey), zap.ByteString("data", data), zap.Error(err))
		return
	}

	logger.Logger.Debug("get websocket msg from delay queue success", zap.String("cacheKey", cacheKey), zap.Any("msg", msg))
	return
}

// 消息收到ACK后保存
func AddMsgAck(userId int, msgId, userConnId string) (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := msgAckPreCacheKey + strconv.Itoa(userId) + ":" + userConnId
	_, err = rd.Do("hSet", cacheKey, msgId, 1)
	if err != nil {
		logger.Logger.Warn("hSet websocket msg ack failed", zap.Int("user_id", userId), zap.String("msg_id", msgId), zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}
	return
}

// 消息收到ACK后删除
func DelMsgAck(userId int, msgId, userConnId string) (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := msgAckPreCacheKey + strconv.Itoa(userId) + ":" + userConnId
	_, err = rd.Do("hDel", cacheKey, msgId)
	if err != nil {
		logger.Logger.Warn("hDel websocket msg ack failed", zap.Int("user_id", userId), zap.String("msg_id", msgId), zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}
	return
}

// 查询消息ACK
func GetMsgAck(userId int, msgId, userConnId string) (ack int, err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := msgAckPreCacheKey + strconv.Itoa(userId) + ":" + userConnId
	ack, err = redis.Int(rd.Do("hGet", cacheKey, msgId))
	if err != nil {
		logger.Logger.Warn("hget websocket msg ack failed", zap.Int("user_id", userId), zap.String("msg_id", msgId), zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}
	return
}

// 消息推送本机的用户链接
func (m Msg) PushMsg(userConnId string) (err error) {
	msg, err := json.Marshal(m)
	if err != nil {
		logger.Logger.Warn("websocket msg json marshal failed", zap.Any("msg", m), zap.Error(err))
		return
	}

	if w, ok := AllWsUserConnInfos[userConnId]; ok {
		if err = w.wsConnection.Send(msg); err != nil {
			logger.Logger.Warn("push websocket msg to user failed", zap.String("user_conn_id", userConnId), zap.Any("msg", m), zap.Error(err))
			return
		}
	}

	logger.Logger.Info("push websocket msg success", zap.Int("user_id", m.UID), zap.Any("msg", msg), zap.String("user_conn_id", m.ConnId))

	return
}

// 推送消息到用户登录的其他服务器
func (m Msg) PushMsgToOtherServer(node, userConnId string) (err error) {
	msg, err := json.Marshal(m)
	if err != nil {
		logger.Logger.Warn("websocket msg json marshal failed", zap.Any("msg", m), zap.Error(err))
		return
	}

	serverUrl := "http://" + node + "/ws/msg/push"
	var data = url.Values{}
	data.Add("conn_id", userConnId)
	data.Add("content", string(msg))

	err = http.Post(serverUrl, data)
	if err != nil {
		logger.Logger.Warn("push websocket msg to other server failed", zap.String("node", node), zap.String("user_id", userConnId), zap.Any("msg", m), zap.Error(err))
		return
	}

	return

}




