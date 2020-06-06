package ws

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

// websocket结构体
type WsConnection struct {
	ID string
	Socket *websocket.Conn
	connection bool
	mu sync.Mutex
}

// 发送消息，加锁，防止分布式多实例并发
func (w *WsConnection) Send(v []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Socket.WriteMessage(websocket.TextMessage, v)
}

// 关闭链接
func (w *WsConnection) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connection = false
	return w.Socket.Close()
}

// 检查链接是否中断
func (w *WsConnection) ping() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Socket.WriteMessage(websocket.PingMessage, nil)
}

// 返回连接是否中断
func (w *WsConnection) IsConnect() bool {
	if w.connection == true {
		if err := w.ping(); err != nil {
			w.connection = false
		}
	}
	return w.connection
}

// 创建链接
func CreateWsConnection(w http.ResponseWriter, r *http.Request) (wsConnction WsConnection, err error) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: time.Second * 5,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	var id = uuid.New().String()
	c, err := upgrader.Upgrade(w, r, http.Header{"uid": {id}})
	if err != nil {
		logger.Logger.Error("CreateWsConnection error", zap.Error(err))
		return
	}

	wsConnction = WsConnection{
		ID:      id,
		Socket:   c,
		connection: true,
	}

	return
}

// 读取链接消息
func (w *WsConnection) Read() (content string, err error) {
	var message []byte
	_, message, err = w.Socket.ReadMessage()
	if err != nil {
		logger.Logger.Error("WsConnection Read message error", zap.Error(err))
		return
	}

	content = string(message)
	return
}



