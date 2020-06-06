package test

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	myredis "go-ws/test/redis"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"strconv"
	"testing"
	"time"
)

const (
	// 消息事件队列前缀
	msgQueuePreCacheKey = "ws_user_msg_queue:"
)

// 消息结构体
type Msg struct {
	ID string `json:"id"`
	UID int `json:"uid"`
	Content interface{} `json:"content"`
	Retries int `json:"retries"`
	ConnId  string `json:"conn_id"`
}

func Benchmark_Ws_SendMessage(b *testing.B) {
	//serverUrl := "http://127.0.0.1:10186/ws/msg/push"
	//userConnId := "86c6ab8c-b132-4f65-aafd-6ec90c227ef5"
	for i := 0; i < b.N; i++ {
		msg := Msg{
			ID:      fmt.Sprintf("%d-%s", time.Now().Unix(), uuid.New().String()),
			UID:     123123,
			Content: "msg::::" + strconv.Itoa(i),
			Retries: 2,
		}

		err := msg.PushWsMsgToQueue()
		if err != nil {
			continue
		}


		logger.Logger.Warn("send websocket msg benchmark", zap.Any("msg", msg), zap.Error(err))

	}
}

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

