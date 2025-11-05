package adapter

import "fmt"

// credKey 生成凭证键
func (r *RedisStorageAdapter) credKey(id string) string {
	return fmt.Sprintf("%s:cred:%s", r.prefix, id)
}

// stateKey 生成状态键
func (r *RedisStorageAdapter) stateKey(id string) string {
	return fmt.Sprintf("%s:state:%s", r.prefix, id)
}

// credSetKey 生成凭证集合键
func (r *RedisStorageAdapter) credSetKey() string {
	return fmt.Sprintf("%s:credentials", r.prefix)
}
