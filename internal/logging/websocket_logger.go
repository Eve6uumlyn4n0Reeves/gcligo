package logging

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// ✅ WebSocketLogger broadcasts log messages to connected WebSocket clients
type WebSocketLogger struct {
	clients         map[*websocket.Conn]*clientInfo
	broadcast       chan LogMessage
	mu              sync.RWMutex
	stopCh          chan struct{}
	history         []LogMessage
	historyMu       sync.RWMutex
	seq             uint64
	historyCap      int
	maxConnections  int
	idleTimeout     time.Duration
	cleanupInterval time.Duration
}

// clientInfo stores metadata about a WebSocket client
type clientInfo struct {
	conn         *websocket.Conn
	lastActivity time.Time
	connected    time.Time
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
		clients:         make(map[*websocket.Conn]*clientInfo),
		broadcast:       make(chan LogMessage, 100),
		stopCh:          make(chan struct{}),
		history:         make([]LogMessage, 0, 500),
		historyCap:      500,
		maxConnections:  100,                // Default max connections
		idleTimeout:     30 * time.Minute,   // Default idle timeout
		cleanupInterval: 2 * time.Minute,    // Default cleanup interval
	}
}

// Start starts the WebSocket logger broadcast service
func (wsl *WebSocketLogger) Start() {
	// Broadcast goroutine
	go func() {
		for {
			select {
			case message := <-wsl.broadcast:
				wsl.mu.RLock()
				for conn, info := range wsl.clients {
					go func(c *websocket.Conn, msg LogMessage) {
						if err := c.WriteJSON(msg); err != nil {
							log.Debugf("Error writing to WebSocket client: %v", err)
							wsl.RemoveClient(c)
						}
					}(conn, message)
					// Update last activity
					info.lastActivity = time.Now()
				}
				wsl.mu.RUnlock()

			case <-wsl.stopCh:
				return
			}
		}
	}()

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(wsl.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				wsl.cleanupDeadConnections()
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

	for conn := range wsl.clients {
		conn.Close()
	}
	wsl.clients = make(map[*websocket.Conn]*clientInfo)
}

// AddClient adds a WebSocket client
func (wsl *WebSocketLogger) AddClient(conn *websocket.Conn) error {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()

	// Check max connections
	if len(wsl.clients) >= wsl.maxConnections {
		log.Warnf("WebSocket connection limit reached (%d), rejecting new connection", wsl.maxConnections)
		return ErrMaxConnectionsReached
	}

	now := time.Now()
	wsl.clients[conn] = &clientInfo{
		conn:         conn,
		lastActivity: now,
		connected:    now,
	}
	log.Infof("WebSocket client connected (total: %d)", len(wsl.clients))
	return nil
}

var ErrMaxConnectionsReached = errors.New("maximum WebSocket connections reached")

// RemoveClient removes a WebSocket client
func (wsl *WebSocketLogger) RemoveClient(conn *websocket.Conn) {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()

	if _, exists := wsl.clients[conn]; exists {
		delete(wsl.clients, conn)
		conn.Close()
		log.Infof("WebSocket client disconnected (remaining: %d)", len(wsl.clients))
	}
}

// cleanupDeadConnections removes idle connections
func (wsl *WebSocketLogger) cleanupDeadConnections() {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()

	now := time.Now()
	toRemove := make([]*websocket.Conn, 0)

	for conn, info := range wsl.clients {
		// Check if connection is idle
		if now.Sub(info.lastActivity) > wsl.idleTimeout {
			toRemove = append(toRemove, conn)
			log.Infof("Removing idle WebSocket connection (idle for %v)", now.Sub(info.lastActivity))
		}
	}

	// Remove idle connections
	for _, conn := range toRemove {
		delete(wsl.clients, conn)
		conn.Close()
	}

	if len(toRemove) > 0 {
		log.Infof("Cleaned up %d idle WebSocket connections (remaining: %d)", len(toRemove), len(wsl.clients))
	}
}

// GetConnectionCount returns the current number of connected clients
func (wsl *WebSocketLogger) GetConnectionCount() int {
	wsl.mu.RLock()
	defer wsl.mu.RUnlock()
	return len(wsl.clients)
}

// SetMaxConnections sets the maximum number of concurrent connections
func (wsl *WebSocketLogger) SetMaxConnections(max int) {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()
	wsl.maxConnections = max
}

// SetIdleTimeout sets the idle timeout duration
func (wsl *WebSocketLogger) SetIdleTimeout(timeout time.Duration) {
	wsl.mu.Lock()
	defer wsl.mu.Unlock()
	wsl.idleTimeout = timeout
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
