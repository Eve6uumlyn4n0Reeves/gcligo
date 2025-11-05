package storage

import (
	"context"
)

// UnsupportedTransactionOps 提供默认的"不支持"事务操作实现
// 用于不支持事务的存储后端（如 MongoDB, Redis）
//
// 使用方法：在后端结构体中嵌入此类型
//
//	type MongoDBBackend struct {
//	    ...
//	    storage.UnsupportedTransactionOps
//	}
type UnsupportedTransactionOps struct{}

// BeginTransaction 返回不支持错误
func (u UnsupportedTransactionOps) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, &ErrNotSupported{Operation: "BeginTransaction"}
}
