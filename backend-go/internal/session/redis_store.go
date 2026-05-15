package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisKeyPrefix  = "ccx:sess:"
	redisRespMapKey = "ccx:respmap"
	redisAllSessKey = "ccx:sess:all"
)

// RedisStore 封装 Redis 操作
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore 创建并测试 Redis 连接
func NewRedisStore(addr, password string, db int, ttl time.Duration) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisStore{client: client, ttl: ttl}, nil
}

// SaveSession 保存会话到 Redis（原子操作：session + active set）
func (s *RedisStore) SaveSession(ctx context.Context, sess *Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	key := redisKeyPrefix + sess.ID
	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, data, s.ttl)
	pipe.SAdd(ctx, redisAllSessKey, sess.ID)
	pipe.Expire(ctx, redisAllSessKey, s.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

// LoadSession 从 Redis 加载会话
func (s *RedisStore) LoadSession(ctx context.Context, sessionID string) (*Session, error) {
	data, err := s.client.Get(ctx, redisKeyPrefix+sessionID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &sess, nil
}

// LookupSessionByResponseID 通过 responseID 查找 sessionID
func (s *RedisStore) LookupSessionByResponseID(ctx context.Context, responseID string) (string, error) {
	sessionID, err := s.client.HGet(ctx, redisRespMapKey, responseID).Result()
	if err == redis.Nil {
		return "", nil
	}
	return sessionID, err
}

// RecordResponseMapping 记录 responseID → sessionID 映射
func (s *RedisStore) RecordResponseMapping(ctx context.Context, responseID, sessionID string) error {
	return s.client.HSet(ctx, redisRespMapKey, responseID, sessionID).Err()
}

// DeleteSession 删除会话
func (s *RedisStore) DeleteSession(ctx context.Context, sessionID string) error {
	pipe := s.client.Pipeline()
	pipe.Del(ctx, redisKeyPrefix+sessionID)
	pipe.SRem(ctx, redisAllSessKey, sessionID)
	_, err := pipe.Exec(ctx)
	return err
}

// GetAllSessionIDs 获取所有活跃会话 ID
func (s *RedisStore) GetAllSessionIDs(ctx context.Context) ([]string, error) {
	return s.client.SMembers(ctx, redisAllSessKey).Result()
}

// GetAllResponseMappings 获取所有映射
func (s *RedisStore) GetAllResponseMappings(ctx context.Context) (map[string]string, error) {
	return s.client.HGetAll(ctx, redisRespMapKey).Result()
}

// CleanOrphanedMappings 清理指向不存在 session 的映射
func (s *RedisStore) CleanOrphanedMappings(ctx context.Context, validSessionIDs map[string]struct{}) error {
	all, err := s.GetAllResponseMappings(ctx)
	if err != nil {
		return err
	}
	pipe := s.client.Pipeline()
	for respID, sessID := range all {
		if _, ok := validSessionIDs[sessID]; !ok {
			pipe.HDel(ctx, redisRespMapKey, respID)
		}
	}
	_, err = pipe.Exec(ctx)
	return err
}

// Close 关闭连接
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// backgroundSave 异步保存，失败只打日志
func (s *RedisStore) backgroundSave(sessionID string, saveFn func(context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := saveFn(ctx); err != nil {
			log.Printf("[Session-Redis] 异步保存失败 (session=%s): %v", sessionID, err)
		}
	}()
}

// SaveSessionAsync 异步保存会话
func (s *RedisStore) SaveSessionAsync(sess *Session) {
	s.backgroundSave(sess.ID, func(ctx context.Context) error {
		return s.SaveSession(ctx, sess)
	})
}

// RecordResponseMappingAsync 异步保存映射
func (s *RedisStore) RecordResponseMappingAsync(responseID, sessionID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.RecordResponseMapping(ctx, responseID, sessionID); err != nil {
			log.Printf("[Session-Redis] 异步保存映射失败 (%s->%s): %v", responseID, sessionID, err)
		}
	}()
}
