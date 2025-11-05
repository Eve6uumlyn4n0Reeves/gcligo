package adapter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorageAdapter Redis存储适配器实现
type RedisStorageAdapter struct {
	client      *redis.Client
	mu          sync.RWMutex
	config      map[string]interface{}
	prefix      string
	keyTTL      time.Duration
	stateKeyTTL time.Duration
	watchers    *watcherHub
}

// RedisStorageConfig Redis存储配置
type RedisStorageConfig struct {
	Addr        string                 `yaml:"addr" json:"addr"`
	Password    string                 `yaml:"password,omitempty" json:"password,omitempty"`
	DB          int                    `yaml:"db" json:"db"`
	Prefix      string                 `yaml:"prefix" json:"prefix"`
	KeyTTL      time.Duration          `yaml:"key_ttl" json:"key_ttl"`
	StateKeyTTL time.Duration          `yaml:"state_key_ttl" json:"state_key_ttl"`
	PoolSize    int                    `yaml:"pool_size" json:"pool_size"`
	Config      map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// NewRedisStorageAdapter 创建新的Redis存储适配器
func NewRedisStorageAdapter(config *RedisStorageConfig) (*RedisStorageAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("redis config cannot be nil")
	}
	if strings.TrimSpace(config.Addr) == "" {
		return nil, fmt.Errorf("redis address cannot be empty")
	}
	if config.Prefix == "" {
		config.Prefix = "gcli2api"
	}
	if config.KeyTTL == 0 {
		config.KeyTTL = 24 * time.Hour
	}
	if config.StateKeyTTL == 0 {
		config.StateKeyTTL = 7 * 24 * time.Hour // 7天
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MaxRetries:   3,
		MinIdleConns: 5,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	adapter := &RedisStorageAdapter{
		client:      client,
		config:      config.Config,
		prefix:      config.Prefix,
		keyTTL:      config.KeyTTL,
		stateKeyTTL: config.StateKeyTTL,
		watchers:    newWatcherHub(),
	}

	return adapter, nil
}

// Ping 检查Redis连接
func (r *RedisStorageAdapter) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close 关闭Redis连接
func (r *RedisStorageAdapter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.client.Close()
}

// GetConfig 获取配置
func (r *RedisStorageAdapter) GetConfig() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config := make(map[string]interface{})
	for k, v := range r.config {
		config[k] = v
	}
	return config
}

// SetConfig 设置配置
func (r *RedisStorageAdapter) SetConfig(ctx context.Context, config map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config
	return nil
}
