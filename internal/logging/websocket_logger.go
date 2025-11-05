package logging

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// ✅ WebSocketLogger broadcasts log messages to connected WebSocket clients
type WebSocketLogger struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan LogMessage
	mu         sync.RWMutex
	stopCh     chan struct{}
	history    []LogMessage
	historyMu  sync.RWMutex
	seq        uint64
	historyCap int
}

// LogMessage represents a log message
type LogMessage struct {
	ID        uint64                 `json:"id,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var (
	globalWSLogger *WebSocketLogger
	wsLoggerOnce   sync.Once
)

// GetWSLogger returns the global WebSocket logger instance
func GetWSLogger() *WebSocketLogger {
	wsLoggerOnce.Do(func() {
		globalWSLogger = NewWebSocketLogger()
		globalWSLogger.Start()
	})
	return globalWSLogger
}

// NewWebSocketLogger creates a new WebSocket logger
func NewWebSocketLogger() *WebSocketLogger {
	return &WebSocketLogger{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan LogMessage, 100),
		stopCh:     make(chan struct{}),
		history:    make([]LogMessage, 0, 500),
		historyCap: 500,
	}
}

// Start starts the WebSocket logger broadcast service
func (wsl *WebSocketLogger) Start() {
	go func() {
		for {
			select {
			case message := <-wsl.broadcast:
				wsl.mu.RLock()
				for client := range wsl.clients {
					go func(c *websocket.Conn, msg LogMessage) {
						if err := c.WriteJSON(msg); err != nil {
							log.Debugf("Error writing to WebSocket client: %v", err)
							wsl.RemoveClient(c)
						}
					}(client, message)
				}
				wsl.mu.RUnlock()

			case <-wsl.stopCh:
				return
			}
		}
	}()
}

// Stop stops the WebSocket logger
func (wsl *WebSocketLogger) Stop() {
	close(wsl.stopCh)

	wsl.mu.Lock()
	defer wsl.mu.Unlock()

	for client := range wsl.clients {
		client.Close()
	}
	wsl.clients = make(map[*websocket.Conn]bool)
}

// AddClient adds a WebSocket client
func (wsl *WebSocketLogger) AddClient(conn *websocket.Conn) {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()

	wsl.clients[conn] = true
	log.Infof("WebSocket client connected (total: %d)", len(wsl.clients))
}

// RemoveClient removes a WebSocket client
func (wsl *WebSocketLogger) RemoveClient(conn *websocket.Conn) {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()

	if wsl.clients[conn] {
		delete(wsl.clients, conn)
		conn.Close()
		log.Infof("WebSocket client disconnected (remaining: %d)", len(wsl.clients))
	}
}

// BroadcastLog broadcasts a log message to all connected clients
func (wsl *WebSocketLogger) BroadcastLog(level, message string, fields map[string]interface{}) {
	id := atomic.AddUint64(&wsl.seq, 1)
	logMsg := LogMessage{
		ID:        id,
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	wsl.appendHistory(logMsg)

	select {
	case wsl.broadcast <- logMsg:
	default:
		// Channel full, drop message
	}
}

func (wsl *WebSocketLogger) appendHistory(msg LogMessage) {
	if wsl.historyCap <= 0 {
		return
	}
	wsl.historyMu.Lock()
	defer wsl.historyMu.Unlock()
	wsl.history = append(wsl.history, msg)
	if len(wsl.history) > wsl.historyCap {
		excess := len(wsl.history) - wsl.historyCap
		wsl.history = append([]LogMessage(nil), wsl.history[excess:]...)
	}
}

// FetchSince returns log messages newer than the provided cursor ID.
func (wsl *WebSocketLogger) FetchSince(cursor uint64, limit int) ([]LogMessage, uint64, bool) {
	wsl.historyMu.RLock()
	defer wsl.historyMu.RUnlock()

	if limit <= 0 || limit > wsl.historyCap {
		limit = wsl.historyCap
	}

	total := len(wsl.history)
	if total == 0 {
		return []LogMessage{}, cursor, false
	}

	start := 0
	if cursor == 0 {
		if total > limit {
			start = total - limit
		}
	} else {
		start = total
		for i, msg := range wsl.history {
			if msg.ID > cursor {
				start = i
				break
			}
		}
		if start >= total {
			return []LogMessage{}, cursor, false
		}
	}

	end := start + limit
	if end > total {
		end = total
	}

	size := end - start
	out := make([]LogMessage, size)
	copy(out, wsl.history[start:end])

	nextCursor := cursor
	if size > 0 {
		nextCursor = out[size-1].ID
	}
	hasMore := end < total

	return out, nextCursor, hasMore
}

// ✅ LogrusHook is a logrus hook that broadcasts to WebSocket clients
type LogrusHook struct {
	wsLogger *WebSocketLogger
}

// NewLogrusHook creates a new logrus hook for WebSocket broadcasting
func NewLogrusHook() *LogrusHook {
	return &LogrusHook{
		wsLogger: GetWSLogger(),
	}
}

// Levels returns the log levels this hook will fire for
func (hook *LogrusHook) Levels() []log.Level {
	return log.AllLevels
}

// Fire is called when a log event occurs
func (hook *LogrusHook) Fire(entry *log.Entry) error {
	fields := make(map[string]interface{})
	for k, v := range entry.Data {
		fields[k] = v
	}

	hook.wsLogger.BroadcastLog(
		entry.Level.String(),
		entry.Message,
		fields,
	)

	return nil
}

// ✅ InstallWebSocketLogging installs WebSocket logging hook
func InstallWebSocketLogging() {
	hook := NewLogrusHook()
	log.AddHook(hook)
	log.Info("WebSocket logging installed")
}
