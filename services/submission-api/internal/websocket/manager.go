package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/online-judge/code-judger/services/submission-api/internal/config"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Manager struct {
	config      config.WebSocketConf
	redisClient *redis.Redis

	// 连接管理
	connections map[string]*Connection
	mutex       sync.RWMutex

	// 消息通道
	broadcast  chan *Message
	register   chan *Connection
	unregister chan *Connection

	// Redis订阅
	redisSub interface{}

	upgrader websocket.Upgrader
}

type Connection struct {
	ID       string
	UserID   int64
	Conn     *websocket.Conn
	Send     chan *Message
	LastPing time.Time
	manager  *Manager
}

type Message struct {
	Type         string      `json:"type"`
	SubmissionID *int64      `json:"submission_id,omitempty"`
	UserID       *int64      `json:"user_id,omitempty"`
	Data         interface{} `json:"data"`
	Timestamp    time.Time   `json:"timestamp"`
}

func NewManager(config config.WebSocketConf, redisClient *redis.Redis) *Manager {
	manager := &Manager{
		config:      config,
		redisClient: redisClient,
		connections: make(map[string]*Connection),
		broadcast:   make(chan *Message, 256),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.BufferSize,
			WriteBufferSize: config.BufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // 在生产环境中应该检查Origin
			},
		},
	}

	// 启动管理协程
	go manager.run()

	// 启动Redis订阅
	go manager.listenRedisMessages()

	// 启动心跳检测
	go manager.heartbeatCheck()

	return manager
}

// run 主运行循环
func (m *Manager) run() {
	for {
		select {
		case conn := <-m.register:
			m.registerConnection(conn)

		case conn := <-m.unregister:
			m.unregisterConnection(conn)

		case message := <-m.broadcast:
			m.broadcastMessage(message)
		}
	}
}

// registerConnection 注册连接
func (m *Manager) registerConnection(conn *Connection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 如果用户已有连接，关闭旧连接
	connKey := fmt.Sprintf("user_%d", conn.UserID)
	if oldConn, exists := m.connections[connKey]; exists {
		close(oldConn.Send)
		oldConn.Conn.Close()
	}

	m.connections[connKey] = conn

	logx.Infof("WebSocket连接注册成功: 用户ID=%d, 连接ID=%s", conn.UserID, conn.ID)

	// 发送欢迎消息
	welcomeMsg := &Message{
		Type:      "welcome",
		Data:      "连接成功",
		Timestamp: time.Now(),
	}

	select {
	case conn.Send <- welcomeMsg:
	default:
		m.unregisterConnection(conn)
	}
}

// unregisterConnection 注销连接
func (m *Manager) unregisterConnection(conn *Connection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	connKey := fmt.Sprintf("user_%d", conn.UserID)
	if _, exists := m.connections[connKey]; exists {
		delete(m.connections, connKey)
		close(conn.Send)
		conn.Conn.Close()

		logx.Infof("WebSocket连接注销: 用户ID=%d, 连接ID=%s", conn.UserID, conn.ID)
	}
}

// broadcastMessage 广播消息
func (m *Manager) broadcastMessage(message *Message) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 如果消息指定了用户ID，只发送给特定用户
	if message.UserID != nil {
		connKey := fmt.Sprintf("user_%d", *message.UserID)
		if conn, exists := m.connections[connKey]; exists {
			select {
			case conn.Send <- message:
			default:
				m.unregister <- conn
			}
		}
		return
	}

	// 如果消息指定了提交ID，需要查找对应的用户
	if message.SubmissionID != nil {
		// 这里应该查询提交记录找到用户ID
		// 为了简化，暂时不实现
		logx.Infof("收到提交状态更新消息: 提交ID=%d", *message.SubmissionID)
		return
	}

	// 广播给所有连接
	for _, conn := range m.connections {
		select {
		case conn.Send <- message:
		default:
			m.unregister <- conn
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID int64) {
	// 升级连接
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("WebSocket升级失败: %v", err)
		return
	}

	// 创建连接对象
	connection := &Connection{
		ID:       fmt.Sprintf("%d_%d", userID, time.Now().Unix()),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan *Message, 256),
		LastPing: time.Now(),
		manager:  m,
	}

	// 注册连接
	m.register <- connection

	// 启动读写协程
	go connection.writePump()
	go connection.readPump()
}

// readPump 读取消息
func (c *Connection) readPump() {
	defer func() {
		c.manager.unregister <- c
	}()

	// 设置读取超时
	c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.manager.config.ReadTimeout) * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.LastPing = time.Now()
		c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.manager.config.ReadTimeout) * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Errorf("WebSocket读取错误: %v", err)
			}
			break
		}
	}
}

// writePump 发送消息
func (c *Connection) writePump() {
	ticker := time.NewTicker(time.Duration(c.manager.config.HeartbeatInterval) * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.manager.config.WriteTimeout) * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				logx.Errorf("WebSocket写入错误: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.manager.config.WriteTimeout) * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// listenRedisMessages 监听Redis消息
func (m *Manager) listenRedisMessages() {
	// 使用Redis List进行简单的消息队列
	// 在实际生产环境中，建议使用专门的消息队列如Kafka
	for {
		// 每隔一段时间检查Redis中的消息
		time.Sleep(time.Second)

		// 使用Lpop获取消息
		result, err := m.redisClient.Lpop("submission_status_updates")
		if err != nil {
			continue // 队列为空或其他错误，继续轮询
		}

		// 解析消息
		var statusUpdate struct {
			SubmissionID int64       `json:"submission_id"`
			UserID       int64       `json:"user_id"`
			Status       string      `json:"status"`
			Data         interface{} `json:"data"`
		}

		if err := json.Unmarshal([]byte(result), &statusUpdate); err != nil {
			logx.Errorf("解析Redis消息失败: %v", err)
			continue
		}

		// 创建WebSocket消息
		wsMessage := &Message{
			Type:         "submission_status_update",
			SubmissionID: &statusUpdate.SubmissionID,
			UserID:       &statusUpdate.UserID,
			Data:         statusUpdate.Data,
			Timestamp:    time.Now(),
		}

		// 广播消息
		m.broadcast <- wsMessage
	}
}

// heartbeatCheck 心跳检测
func (m *Manager) heartbeatCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		for key, conn := range m.connections {
			if time.Since(conn.LastPing) > time.Duration(m.config.HeartbeatInterval*2)*time.Second {
				// 超时连接，主动关闭
				logx.Infof("WebSocket连接超时，主动关闭: %s", key)
				m.unregister <- conn
			}
		}
		m.mutex.Unlock()
	}
}

// PublishStatusUpdate 发布状态更新到Redis
func (m *Manager) PublishStatusUpdate(submissionID, userID int64, status string, data interface{}) error {
	message := map[string]interface{}{
		"submission_id": submissionID,
		"user_id":       userID,
		"status":        status,
		"data":          data,
		"timestamp":     time.Now().Unix(),
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = m.redisClient.Lpush("submission_status_updates", string(payload))
	return err
}

// GetConnectionCount 获取当前连接数
func (m *Manager) GetConnectionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.connections)
}
