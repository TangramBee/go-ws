package wsservice

import (
	"encoding/json"
	myredis "go-ws/databases/redis"
	"go-ws/utils/logger"
	"go-ws/utils/ws"
	"go.uber.org/zap"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	// 用户的链接ID列表
	wsUserConnectionListPreCacheKey =  "ws_user_connection_list:"
	// 在线的用户ID列表
	wsUserOnlineListKey = "ws_user_online_list"
	// 用户链接数据
	WsUserConnInfoPreCacheKey = "ws_user_info:"
)

// 用户链接数据结构体
type WsUserConnInfo struct {
	ID  string `json:"id"`
	UID int `json:"uid"`
	Node string `json:"node"`
	Closed         bool   `json:"closed"`
	ConnectTime    int64  `json:"connect_time"`
	DisConnectTime int64  `json:"disconnect_time"`
	wsConnection *ws.WsConnection
	mu   sync.Mutex
	messages chan []byte
}

var (
	AllWsUserConnInfos = make(map[string]*WsUserConnInfo)
	addWsUserConnInfos chan *WsUserConnInfo
	delWsUserConnInfos chan *WsUserConnInfo
)

// 管理用户链接
func ManagerWsUserConnInfos()  {
	// 关闭原来的链接
	CloseAllWsUserConn()

	addWsUserConnInfos = make(chan *WsUserConnInfo)
	delWsUserConnInfos = make(chan *WsUserConnInfo)

	for {
		select {
		case w := <- addWsUserConnInfos:
			// 添加到本机的户链接的映射关系map
			AllWsUserConnInfos[w.ID] = w
			// 更新用户链接信息
			w.UpdateUserInfo()
			// 添加用户ID
			AddOnlineUserId(w.UID)
			// 添加用户链接ID
			AddWsUserConnId(w.UID, w.ID)
		case w := <- delWsUserConnInfos:
			w.wsConnection.Close()

			// 加锁，防止启动多个项目实例并发处理
			w.mu.Lock()
			if !w.Closed {
				w.Closed = true
				w.DisConnectTime = time.Now().Unix()

				close(w.messages)
				// 更新用户链接信息
				w.UpdateUserInfo()
				// 删除本机用户链接的映射关系map
				DelLocalUserConn(w.ID)
				// 删除用户链接信息
				go w.DelUserInfo()

			}
			w.mu.Unlock()
		}
	}
}

// 更新用户信息
func (w *WsUserConnInfo) UpdateUserInfo() (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := WsUserConnInfoPreCacheKey + w.ID

	_, err = rd.Do("hMSet", redis.Args{}.Add(cacheKey).AddFlat(w)...)
	if err != nil {
		logger.Logger.Warn("update user info failed", zap.Any("user_info", w), zap.Error(err))
		return
	}
	return
}

// 删除用户数据，通过设置过期时间让redis自动删除,由于命令是单线程执行，这样避免过多del删除操作影响其他命令
func (w *WsUserConnInfo) DelUserInfo() (err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := WsUserConnInfoPreCacheKey + w.ID
	_, err = rd.Do("expire", cacheKey, 86400)
	if err != nil {
		logger.Logger.Warn("del user info failed", zap.Any("user_info", w), zap.Error(err))
		return
	}
	return
}

// 心跳检测
func (w *WsUserConnInfo) HeartBeatCheck() {
	defer func() {
		delWsUserConnInfos <- w
	}()
	for {
		select {
		// 1秒没有消息就检测是否断连
		case <- time.After(time.Millisecond * 10): 
			if !w.wsConnection.IsConnect() {
				logger.Logger.Warn("ping websocket user conn failed", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID))
				return
			}
		}
	}
}

// 循环推送队列消息给用户
func (w *WsUserConnInfo) PushLoop() {
	// 启动链接的时候检测是否有消息未成功推送
	go w.MsgAckDelayCheck()

	// 监听推送消息
	for {
		// 判断是否断开链接
		if !w.wsConnection.IsConnect() {
			logger.Logger.Warn("user websocket disconnect", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID))
			break
		}

		msg, err := PopWsMsgFromQueue(w.UID)
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		msg.ConnId = w.ID

		// 向某个用户的所有链接同步推送消息
		go func(userId int, msg Msg) {
			var userConnIdList []WsUserConnInfo
			userConnIdList, err = GetAllUserInfoList(userId)
			if err != nil {
				logger.Logger.Warn("get user all websocket conn id failed", zap.Int("user_id", userId), zap.Any("msg", msg), zap.Error(err))
				return
			}

			for _, userConn := range userConnIdList {
				if userConn.Closed {
					continue
				}
				if _, ok := AllWsUserConnInfos[userConn.ID]; ok {
					go msg.PushMsg(userConn.ID)
				} else {
					go msg.PushMsgToOtherServer(userConn.Node, userConn.ID)
				}

				// 记录消息到延迟队列，判断用户是否收到消息ACK
				if msg.Retries > 0 {
					msg.ConnId = userConn.ID
					go msg.PushWsMsgToDelayQueue()
				}
			}

		}(w.UID, msg)

		logger.Logger.Info("send websocket msg success", zap.Int("user_id", w.UID), zap.Any("msg", msg), zap.String("user_conn_id", w.ID))
	}
}

// 循环接受发送给用户的消息
func (w *WsUserConnInfo) ReceiveLoop() {
	for {
		// 判断是否断开链接
		if !w.wsConnection.IsConnect() {
			break
		}

		recMsgStr, err := w.wsConnection.Read()
		if err != nil {
			logger.Logger.Warn("receive websocket msg failed", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID), zap.Error(err))
			continue
		}

		var recMsg RecMsg
		if err = json.Unmarshal([]byte(recMsgStr), recMsg); err != nil {
			logger.Logger.Warn("receive websocket msg json unmarshal failed", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID), zap.String("receive_msg", recMsgStr), zap.Error(err))
			continue
		}

		if recMsg.ID != "" {
			// 保存消息的ack
			_ = AddMsgAck(w.UID, recMsg.ID, w.ID)
		}

		logger.Logger.Info("receive websocket msg success", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID), zap.String("receive_msg", recMsgStr), zap.Error(err))

	}
}

// 消息延迟检测ACK
func (w *WsUserConnInfo) MsgAckDelayCheck() {
	for {
		if !w.wsConnection.IsConnect() {
			logger.Logger.Warn("delay check websocket user disconnect failed", zap.Int("user_id", w.UID), zap.String("user_conn_id", w.ID))
			break
		}

		msg, err := PopWsMsgFromDelayQueue(w.UID)
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		// 判断之前发送的客户端有收到ACK，删除ACK记录，如没有收到消息的ACK记录会返回nil报错
		_, err = GetMsgAck(w.UID, msg.ID, w.ID)
		if err == nil {
			go DelMsgAck(w.UID, msg.ID, w.ID)
			continue
		}

		// 没有收到ACK，就再发一次
		if userConn, ok := AllWsUserConnInfos[msg.ConnId]; ok {
			if msg.Retries > 0 {
				msg.Retries = msg.Retries - 1
				_ = msg.PushMsg(userConn.ID)
				// 记录消息到延迟队列，判断用户是否收到消息ACK
				go msg.PushWsMsgToDelayQueue()
			}
		} else {
			// 查找客户端
			if userConn, err := GetWsUserConnInfo(userConn.ID); err != nil {
				if userConn.Closed == false {
					continue
				}
				if msg.Retries > 0 {
					msg.Retries = msg.Retries - 1
					go msg.PushMsgToOtherServer(userConn.Node, userConn.ID)
					// 记录消息到延迟队列，判断用户是否收到消息ACK
					go msg.PushWsMsgToDelayQueue()
				}
			}
		}
		
	}
}


// 添加用户链接信息
func AddWsUserConnInfo(userId int, node string, w *ws.WsConnection) WsUserConnInfo {
	u := WsUserConnInfo{
		ID:             w.ID,
		UID:            userId,
		Node:           node,
		Closed:         false,
		ConnectTime:    time.Now().Unix(),
		DisConnectTime: 0,
		wsConnection:   w,
		messages:       make(chan []byte, 1000),
	}

	logger.Logger.Info("add websocket user info success", zap.Int("user_id", userId), zap.String("user_conn_id", w.ID), zap.String("node", node))

	addWsUserConnInfos <- &u
	return u
}

// 关闭所有客户端的链接
func CloseAllWsUserConn() (err error) {
	var onlineUserIdList []int
	// 获取用户ID列表
	onlineUserIdList, err = GetOnLineUserIdList()
	if err != nil {
		logger.Logger.Warn("get websocket user list failed", zap.Error(err))
		return
	}

	for _, onlineUserId := range onlineUserIdList {
		var userConnIdList []string
		// 获取用户链接ID列表
		userConnIdList, err = GetWsUserConnIdList(onlineUserId)
		if err != nil {
			logger.Logger.Warn("get online user id list failed", zap.Int("user_id", onlineUserId), zap.Error(err))
			continue
		}

		for _, userConnId := range userConnIdList {
			go func(onlineUserId int, userConnId string) {
				var w WsUserConnInfo
				// 获取用户信息
				w, err = GetWsUserConnInfo(userConnId)
				if err != nil {
					// 查不到时删除列表里的clientID
					if err == redis.ErrNil {
						// 删除用户链接ID
						go DelWsUserConnId(onlineUserId, userConnId)
					}
					return
				}

				// 设置redis中连接为关闭状态
				if w.Closed == false {
					w.Closed = true
					w.DisConnectTime = time.Now().Unix()
					// 更新用户链接信息
					w.UpdateUserInfo()
					// 删除用户链接信息
					go w.DelUserInfo()
				}

				// 删除已关闭7天的连接
				if w.DisConnectTime <= time.Now().AddDate(0, 0, -7).Unix() {
					// 删除用户链接ID
					DelWsUserConnId(onlineUserId, userConnId)
				}

			}(onlineUserId, userConnId)
		}

	}

	return
}

// 获取用户链接数据
func GetWsUserConnInfo(userConnId string) (w WsUserConnInfo, err error) {
	rd := myredis.NewRedis("default_redis").Get()

	cacheKey := WsUserConnInfoPreCacheKey + userConnId
	var v []interface{}
	v, err = redis.Values(rd.Do("hGetAll", cacheKey))
	if err != nil {
		logger.Logger.Warn("get websocket user info failed", zap.String("user_conn_id", userConnId), zap.Error(err))
		return
	}

	err = redis.ScanStruct(v, &w)
	if err != nil {
		logger.Logger.Warn("scan websocket user info failed", zap.String("user_conn_id", userConnId), zap.Any("user_info", v), zap.Error(err))
		return
	}

	return
}

// 获取某个用户的所有链接信息，推送用户消息
func GetAllUserInfoList(userId int) (w []WsUserConnInfo, err error) {
	var userConnIdList []string
	userConnIdList, err = GetWsUserConnIdList(userId)
	if err != nil {
		logger.Logger.Warn("get websocket user conn id list failed", zap.Int("user_id", userId), zap.Error(err))
		return
	}

	var wg sync.WaitGroup
	var ch = make(chan WsUserConnInfo, len(userConnIdList))

	for _, userConnId := range userConnIdList {
		wg.Add(1)

		go func(userConnId string) {
			defer wg.Done()
			if w, err := GetWsUserConnInfo(userConnId); err == nil {

				ch <- w
			}
		}(userConnId)
	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	for c := range ch {
		w = append(w, c)
	}

	return

}

