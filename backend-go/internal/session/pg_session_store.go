package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGSessionStore PG 持久化 session 存储
type PGSessionStore struct {
	pool *pgxpool.Pool
	ttl  time.Duration // 用于清理查询，非 PG-side TTL
}

// NewPGSessionStore 创建并初始化 PG session 存储
func NewPGSessionStore(databaseURL string, ttl time.Duration) (*PGSessionStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PG: %w", err)
	}

	store := &PGSessionStore{pool: pool, ttl: ttl}
	if err := store.ensureTables(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensure tables: %w", err)
	}

	return store, nil
}

func (s *PGSessionStore) ensureTables() error {
	_, err := s.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			data JSONB NOT NULL,
			last_access_at BIGINT NOT NULL,
			created_at BIGINT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_last_access ON sessions(last_access_at);
		CREATE TABLE IF NOT EXISTS response_map (
			response_id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_response_map_session ON response_map(session_id);
	`)
	return err
}

// SaveSession 保存会话到 PG（upsert）
func (s *PGSessionStore) SaveSession(ctx context.Context, sess *Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO sessions (session_id, data, last_access_at, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id) DO UPDATE SET
			data = EXCLUDED.data,
			last_access_at = EXCLUDED.last_access_at`,
		sess.ID, data, sess.LastAccessAt.Unix(), sess.CreatedAt.Unix(),
	)
	return err
}

// LoadSession 从 PG 加载会话
func (s *PGSessionStore) LoadSession(ctx context.Context, sessionID string) (*Session, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		"SELECT data FROM sessions WHERE session_id = $1", sessionID).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // 不存在返回 nil
		}
		return nil, fmt.Errorf("load session from PG: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &sess, nil
}

// LookupSessionByResponseID 通过 responseID 查找 sessionID
func (s *PGSessionStore) LookupSessionByResponseID(ctx context.Context, responseID string) (string, error) {
	var sessionID string
	err := s.pool.QueryRow(ctx,
		"SELECT session_id FROM response_map WHERE response_id = $1", responseID).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // 不存在返回空
		}
		return "", fmt.Errorf("lookup session by response: %w", err)
	}
	return sessionID, nil
}

// RecordResponseMapping 记录映射
func (s *PGSessionStore) RecordResponseMapping(ctx context.Context, responseID, sessionID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO response_map (response_id, session_id) VALUES ($1, $2)
		ON CONFLICT (response_id) DO UPDATE SET session_id = EXCLUDED.session_id`,
		responseID, sessionID,
	)
	return err
}

// DeleteSession 删除会话（事务保证原子性）
func (s *PGSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		"DELETE FROM response_map WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("delete response_map: %w", err)
	}
	_, err = tx.Exec(ctx,
		"DELETE FROM sessions WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return tx.Commit(ctx)
}

// GetAllSessionIDs 获取所有会话 ID
func (s *PGSessionStore) GetAllSessionIDs(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, "SELECT session_id FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetAllResponseMappings 获取所有映射
func (s *PGSessionStore) GetAllResponseMappings(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, "SELECT response_id, session_id FROM response_map")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var respID, sessID string
		if err := rows.Scan(&respID, &sessID); err != nil {
			return nil, err
		}
		result[respID] = sessID
	}
	return result, rows.Err()
}

// CleanOrphanedMappings 清理指向不存在会话的映射
func (s *PGSessionStore) CleanOrphanedMappings(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM response_map
		WHERE session_id NOT IN (SELECT session_id FROM sessions)`)
	return err
}

// CleanupExpiredSessions 清理过期会话（PG-side）
// 当 maxMessages > 0 或 maxTokens > 0 时，同时清理消息数/Token 超限的会话。
func (s *PGSessionStore) CleanupExpiredSessions(ctx context.Context, maxAge time.Duration, maxMessages int, maxTokens int) (int64, error) {
	cutoff := time.Now().Add(-maxAge).Unix()

	conditions := []string{"last_access_at < $1"}
	args := []interface{}{cutoff}
	argIdx := 2

	if maxMessages > 0 {
		conditions = append(conditions, fmt.Sprintf("jsonb_array_length(data->'messages') > $%d", argIdx))
		args = append(args, maxMessages)
		argIdx++
	}
	if maxTokens > 0 {
		conditions = append(conditions, fmt.Sprintf("(data->>'total_tokens')::int > $%d", argIdx))
		args = append(args, maxTokens)
		argIdx++
	}

	query := fmt.Sprintf("DELETE FROM sessions WHERE %s", strings.Join(conditions, " OR "))

	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	// 同时清理孤立映射
	s.pool.Exec(ctx, `
		DELETE FROM response_map
		WHERE session_id NOT IN (SELECT session_id FROM sessions)`)
	return tag.RowsAffected(), nil
}

// SaveSessionAsync 异步保存
func (s *PGSessionStore) SaveSessionAsync(sess *Session) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.SaveSession(ctx, sess); err != nil {
			log.Printf("[Session-PG] 异步保存失败 (session=%s): %v", sess.ID, err)
		}
	}()
}

// RecordResponseMappingAsync 异步保存映射
func (s *PGSessionStore) RecordResponseMappingAsync(responseID, sessionID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.RecordResponseMapping(ctx, responseID, sessionID); err != nil {
			log.Printf("[Session-PG] 异步保存映射失败 (%s->%s): %v", responseID, sessionID, err)
		}
	}()
}

// Close 关闭连接
func (s *PGSessionStore) Close() {
	s.pool.Close()
}
