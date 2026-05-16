package metrics

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 硬编码的内部配置
const (
	defaultBatchSize     = 100
	defaultFlushInterval = 30 * time.Second
)

// 迁移相关类型
type metricsKeyMigrationTarget struct {
	MetricsKey string
	BaseURL    string
}

type metricsKeyMigrationCandidates struct {
	Primary   metricsKeyMigrationTarget
	Conflicts []metricsKeyMigrationTarget
}

type persistedCircuitStateRow struct {
	PersistentCircuitState
	UpdatedAt int64
}

// PostgresStore Postgres 持久化存储
type PostgresStore struct {
	pool *pgxpool.Pool

	// 写入缓冲区
	writeBuffer []PersistentRecord
	bufferMu    sync.Mutex

	// 配置
	batchSize     int           // 批量写入阈值（记录数）
	flushInterval time.Duration // 定时刷新间隔
	retentionDays int           // 数据保留天数

	// 控制
	stopCh       chan struct{}
	wg           sync.WaitGroup
	closed       bool           // 是否已关闭
	flushMu      sync.Mutex     // 串行化 flush 与 delete 操作，避免并发竞态
	asyncFlushWg sync.WaitGroup // 追踪 AddRecord 触发的异步 flush goroutine
	flushing     atomic.Bool    // 原子标记：是否有 flush goroutine 正在运行/排队
}

// PostgresStoreConfig Postgres 存储配置
type PostgresStoreConfig struct {
	DatabaseURL   string
	RetentionDays int
}

// NewPostgresStore 创建 Postgres 存储
func NewPostgresStore(cfg *PostgresStoreConfig) (*PostgresStore, error) {
	if cfg == nil {
		cfg = &PostgresStoreConfig{
			RetentionDays: 30,
		}
	}

	// 验证保留天数范围
	retentionDays := cfg.RetentionDays
	if retentionDays < 3 {
		retentionDays = 3
	} else if retentionDays > 90 {
		retentionDays = 90
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("创建连接池失败: %w", err)
	}

	// 验证连接
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 初始化表结构
	if err := initPostgresSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("初始化数据库 schema 失败: %w", err)
	}

	store := &PostgresStore{
		pool:          pool,
		writeBuffer:   make([]PersistentRecord, 0, defaultBatchSize),
		batchSize:     defaultBatchSize,
		flushInterval: defaultFlushInterval,
		retentionDays: retentionDays,
		stopCh:        make(chan struct{}),
	}

	// 启动后台任务
	store.wg.Add(2)
	go store.flushLoop()
	go store.cleanupLoop()

	log.Printf("[Postgres-Init] 指标存储已初始化 (保留 %d 天)", retentionDays)
	return store, nil
}

// initPostgresSchema 初始化 Postgres 数据库表结构
func initPostgresSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
		CREATE TABLE IF NOT EXISTS schema_version (
			id INTEGER PRIMARY KEY DEFAULT 1,
			version INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS request_records (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			metrics_key TEXT NOT NULL,
			base_url TEXT NOT NULL,
			key_mask TEXT NOT NULL,
			timestamp BIGINT NOT NULL,
			success BOOLEAN NOT NULL,
			failure_class TEXT NOT NULL DEFAULT '',
			input_tokens BIGINT DEFAULT 0,
			output_tokens BIGINT DEFAULT 0,
			cache_creation_tokens BIGINT DEFAULT 0,
			cache_read_tokens BIGINT DEFAULT 0,
			api_type TEXT NOT NULL DEFAULT 'messages',
			model TEXT DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS circuit_states (
			metrics_key TEXT NOT NULL,
			api_type TEXT NOT NULL,
			base_url TEXT NOT NULL,
			key_mask TEXT NOT NULL,
			circuit_state TEXT NOT NULL DEFAULT 'closed',
			circuit_opened_at BIGINT,
			half_open_at BIGINT,
			next_retry_at BIGINT,
			backoff_level INTEGER NOT NULL DEFAULT 0,
			half_open_successes INTEGER NOT NULL DEFAULT 0,
			consecutive_failures BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (metrics_key, api_type)
		);

		CREATE INDEX IF NOT EXISTS idx_records_api_type_timestamp
			ON request_records(api_type, timestamp);

		CREATE INDEX IF NOT EXISTS idx_records_metrics_key
			ON request_records(metrics_key);

		CREATE INDEX IF NOT EXISTS idx_records_model
			ON request_records(model);
	`

	if _, err := pool.Exec(ctx, schema); err != nil {
		return err
	}

	// 确保 schema_version 表有初始行
	if _, err := pool.Exec(ctx, `
		INSERT INTO schema_version (id, version)
		VALUES (1, 0)
		ON CONFLICT (id) DO NOTHING
	`); err != nil {
		return err
	}

	// 版本迁移：使用 schema_version 表检测版本
	var version int
	if err := pool.QueryRow(ctx, "SELECT version FROM schema_version WHERE id = 1").Scan(&version); err != nil {
		return err
	}

	if version < 1 {
		migrations := []string{
			"ALTER TABLE request_records ADD COLUMN IF NOT EXISTS model TEXT DEFAULT ''",
			"UPDATE schema_version SET version = 1 WHERE id = 1",
		}
		for _, sql := range migrations {
			if _, err := pool.Exec(ctx, sql); err != nil {
				return fmt.Errorf("migration v0->v1 failed: %w", err)
			}
		}
		log.Printf("[Postgres-Migration] schema 升级: v0 -> v1 (添加 model 列)")
		version = 1
	}

	if version < 2 {
		migrations := []string{
			"ALTER TABLE request_records ADD COLUMN IF NOT EXISTS failure_class TEXT NOT NULL DEFAULT ''",
			"UPDATE schema_version SET version = 2 WHERE id = 1",
		}
		for _, sql := range migrations {
			if _, err := pool.Exec(ctx, sql); err != nil {
				return fmt.Errorf("migration v1->v2 failed: %w", err)
			}
		}
		log.Printf("[Postgres-Migration] schema 升级: v1 -> v2 (添加 failure_class 列与 circuit_states 表)")
	}

	return nil
}

func (s *PostgresStore) pgSchemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := s.pool.QueryRow(ctx, "SELECT version FROM schema_version WHERE id = 1").Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func (s *PostgresStore) MigrateMetricsKeysToIdentity(cfg config.Config) error {
	ctx := context.Background()
	version, err := s.pgSchemaVersion(ctx)
	if err != nil {
		return fmt.Errorf("读取 schema 版本失败: %w", err)
	}
	if version >= 3 {
		return nil
	}

	mapping := buildMetricsKeyMigrationMap(cfg)

	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	s.flushBufferLocked()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始 metrics key 迁移事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	updatedRecords, err := migrateRequestRecordsTxPG(ctx, tx, mapping)
	if err != nil {
		return err
	}
	migratedStates, mergedStates, err := migrateCircuitStatesTxPG(ctx, tx, mapping)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE schema_version SET version = 3 WHERE id = 1"); err != nil {
		return fmt.Errorf("写入 schema 版本失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交 metrics key 迁移失败: %w", err)
	}

	log.Printf("[Postgres-Migration] schema/data 升级: v2 -> v3 (迁移 request_records=%d, circuit_states=%d, merged_states=%d)", updatedRecords, migratedStates, mergedStates)
	return nil
}

func migrateRequestRecordsTxPG(ctx context.Context, tx pgx.Tx, mapping map[string]map[string]metricsKeyMigrationCandidates) (int64, error) {
	var totalUpdated int64
	for apiType, targets := range mapping {
		for legacyKey, candidate := range targets {
			if len(candidate.Conflicts) > 0 {
				var count int
				if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM request_records WHERE api_type = $1 AND metrics_key = $2`, apiType, legacyKey).Scan(&count); err != nil {
					return totalUpdated, fmt.Errorf("检查 request_records 迁移冲突失败(apiType=%s, metricsKey=%s): %w", apiType, legacyKey, err)
				}
				if count > 0 {
					return totalUpdated, fmt.Errorf("legacy metrics key %s 在 apiType=%s 的 request_records 中存在 %d 条待迁移记录，但映射到多个 identity target: primary=%+v conflicts=%+v", legacyKey, apiType, count, candidate.Primary, candidate.Conflicts)
				}
			}
			result, err := tx.Exec(ctx, `
				UPDATE request_records
				SET metrics_key = $1, base_url = $2
				WHERE api_type = $3 AND metrics_key = $4 AND (metrics_key != $5 OR base_url != $6)
			`, candidate.Primary.MetricsKey, candidate.Primary.BaseURL, apiType, legacyKey, candidate.Primary.MetricsKey, candidate.Primary.BaseURL)
			if err != nil {
				return totalUpdated, fmt.Errorf("迁移 request_records 失败(apiType=%s, metricsKey=%s): %w", apiType, legacyKey, err)
			}
			totalUpdated += result.RowsAffected()
		}
	}
	return totalUpdated, nil
}

func migrateCircuitStatesTxPG(ctx context.Context, tx pgx.Tx, mapping map[string]map[string]metricsKeyMigrationCandidates) (int, int, error) {
	rows, err := tx.Query(ctx, `
		SELECT metrics_key, api_type, base_url, key_mask, circuit_state,
		       circuit_opened_at, half_open_at, next_retry_at,
		       backoff_level, half_open_successes, consecutive_failures, updated_at
		FROM circuit_states
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("查询 circuit_states 失败: %w", err)
	}
	defer rows.Close()

	merged := make(map[string]*persistedCircuitStateRow)
	migratedCount := 0
	mergedCount := 0
	for rows.Next() {
		var row persistedCircuitStateRow
		var openedAt, halfOpenAt, nextRetryAt *int64
		if err := rows.Scan(
			&row.MetricsKey,
			&row.APIType,
			&row.BaseURL,
			&row.KeyMask,
			&row.CircuitState,
			&openedAt,
			&halfOpenAt,
			&nextRetryAt,
			&row.BackoffLevel,
			&row.HalfOpenSuccesses,
			&row.ConsecutiveFailures,
			&row.UpdatedAt,
		); err != nil {
			return migratedCount, mergedCount, fmt.Errorf("扫描 circuit_states 失败: %w", err)
		}
		if openedAt != nil {
			t := time.Unix(*openedAt, 0)
			row.CircuitOpenedAt = &t
		}
		if halfOpenAt != nil {
			t := time.Unix(*halfOpenAt, 0)
			row.HalfOpenAt = &t
		}
		if nextRetryAt != nil {
			t := time.Unix(*nextRetryAt, 0)
			row.NextRetryAt = &t
		}

		if candidate, ok := mapping[row.APIType][row.MetricsKey]; ok {
			if len(candidate.Conflicts) > 0 {
				return migratedCount, mergedCount, fmt.Errorf("legacy metrics key %s 在 apiType=%s 的 circuit_states 中存在待迁移记录，但映射到多个 identity target: primary=%+v conflicts=%+v", row.MetricsKey, row.APIType, candidate.Primary, candidate.Conflicts)
			}
			if row.MetricsKey != candidate.Primary.MetricsKey || row.BaseURL != candidate.Primary.BaseURL {
				migratedCount++
			}
			row.MetricsKey = candidate.Primary.MetricsKey
			row.BaseURL = candidate.Primary.BaseURL
		}

		mergedKey := row.APIType + "|" + row.MetricsKey
		if existing, exists := merged[mergedKey]; exists {
			mergePersistedCircuitState(existing, &row)
			mergedCount++
			continue
		}
		copyRow := row
		merged[mergedKey] = &copyRow
	}
	if err := rows.Err(); err != nil {
		return migratedCount, mergedCount, fmt.Errorf("遍历 circuit_states 失败: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM circuit_states"); err != nil {
		return migratedCount, mergedCount, fmt.Errorf("清空 circuit_states 失败: %w", err)
	}

	for _, row := range merged {
		var openedAt any
		if row.CircuitOpenedAt != nil {
			openedAt = row.CircuitOpenedAt.Unix()
		}
		var halfOpenAt any
		if row.HalfOpenAt != nil {
			halfOpenAt = row.HalfOpenAt.Unix()
		}
		var nextRetryAt any
		if row.NextRetryAt != nil {
			nextRetryAt = row.NextRetryAt.Unix()
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO circuit_states (
				metrics_key, api_type, base_url, key_mask, circuit_state,
				circuit_opened_at, half_open_at, next_retry_at,
				backoff_level, half_open_successes, consecutive_failures, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, row.MetricsKey, row.APIType, row.BaseURL, row.KeyMask, row.CircuitState, openedAt, halfOpenAt, nextRetryAt, row.BackoffLevel, row.HalfOpenSuccesses, row.ConsecutiveFailures, time.Now().Unix()); err != nil {
			return migratedCount, mergedCount, fmt.Errorf("重建 circuit_states 失败(metricsKey=%s, apiType=%s): %w", row.MetricsKey, row.APIType, err)
		}
	}

	return migratedCount, mergedCount, nil
}

// AddRecord 添加记录到写入缓冲区（非阻塞）
func (s *PostgresStore) AddRecord(record PersistentRecord) {
	s.bufferMu.Lock()
	if s.closed {
		s.bufferMu.Unlock()
		return // 已关闭，忽略新记录
	}
	s.writeBuffer = append(s.writeBuffer, record)
	shouldFlush := len(s.writeBuffer) >= s.batchSize
	s.bufferMu.Unlock()

	// 使用原子标记确保同一时间只有一个 flush goroutine 被调度
	// 避免高并发下产生大量 goroutine 排队等待 flushMu
	if shouldFlush && s.flushing.CompareAndSwap(false, true) {
		s.asyncFlushWg.Add(1)
		go func() {
			defer s.asyncFlushWg.Done()
			defer s.flushing.Store(false)
			// 获取 flush 锁，与 DeleteRecordsByMetricsKeys 串行化
			s.flushMu.Lock()
			s.flush()
			s.flushMu.Unlock()
		}()
	}
}

// flush 刷新缓冲区到数据库
func (s *PostgresStore) flush() {
	s.bufferMu.Lock()
	if len(s.writeBuffer) == 0 {
		s.bufferMu.Unlock()
		return
	}

	// 取出缓冲区数据
	records := s.writeBuffer
	s.writeBuffer = make([]PersistentRecord, 0, s.batchSize)
	s.bufferMu.Unlock()

	// 批量写入
	if err := s.batchInsertRecords(records); err != nil {
		log.Printf("[Postgres-Flush] 警告: 批量写入指标记录失败: %v", err)
		s.requeueRecords(records, "[Postgres-Flush]")
	}
}

// batchInsertRecords 批量插入记录
func (s *PostgresStore) batchInsertRecords(records []PersistentRecord) error {
	if len(records) == 0 {
		return nil
	}

	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, r := range records {
		_, err := tx.Exec(ctx, `
			INSERT INTO request_records
			(metrics_key, base_url, key_mask, timestamp, success, failure_class,
			 input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens, api_type, model)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, r.MetricsKey, r.BaseURL, r.KeyMask, r.Timestamp.Unix(), r.Success, string(r.FailureClass),
			r.InputTokens, r.OutputTokens, r.CacheCreationTokens, r.CacheReadTokens, r.APIType, r.Model)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// LoadRecords 加载指定时间范围内的记录
func (s *PostgresStore) LoadRecords(since time.Time, apiType string) ([]PersistentRecord, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `
		SELECT metrics_key, base_url, key_mask, timestamp, success, failure_class,
		       input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens, model
		FROM request_records
		WHERE timestamp >= $1 AND api_type = $2
		ORDER BY timestamp ASC
	`, since.Unix(), apiType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PersistentRecord
	for rows.Next() {
		var r PersistentRecord
		var ts int64
		var failureClass string

		err := rows.Scan(
			&r.MetricsKey, &r.BaseURL, &r.KeyMask, &ts, &r.Success, &failureClass,
			&r.InputTokens, &r.OutputTokens, &r.CacheCreationTokens, &r.CacheReadTokens, &r.Model,
		)
		if err != nil {
			return nil, err
		}

		r.Timestamp = time.Unix(ts, 0)
		r.FailureClass = FailureClass(failureClass)
		r.APIType = apiType
		records = append(records, r)
	}

	return records, rows.Err()
}

// LoadCircuitStates 加载指定 API 类型的 breaker 状态。
func (s *PostgresStore) LoadCircuitStates(apiType string) (map[string]*PersistentCircuitState, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `
		SELECT metrics_key, base_url, key_mask, circuit_state,
		       circuit_opened_at, half_open_at, next_retry_at,
		       backoff_level, half_open_successes, consecutive_failures
		FROM circuit_states
		WHERE api_type = $1
	`, apiType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*PersistentCircuitState)
	for rows.Next() {
		var state PersistentCircuitState
		var openedAt, halfOpenAt, nextRetryAt *int64
		if err := rows.Scan(
			&state.MetricsKey,
			&state.BaseURL,
			&state.KeyMask,
			&state.CircuitState,
			&openedAt,
			&halfOpenAt,
			&nextRetryAt,
			&state.BackoffLevel,
			&state.HalfOpenSuccesses,
			&state.ConsecutiveFailures,
		); err != nil {
			return nil, err
		}
		state.APIType = apiType
		if openedAt != nil {
			t := time.Unix(*openedAt, 0)
			state.CircuitOpenedAt = &t
		}
		if halfOpenAt != nil {
			t := time.Unix(*halfOpenAt, 0)
			state.HalfOpenAt = &t
		}
		if nextRetryAt != nil {
			t := time.Unix(*nextRetryAt, 0)
			state.NextRetryAt = &t
		}
		result[state.MetricsKey] = &state
	}

	return result, rows.Err()
}

// UpsertCircuitState 写入或更新 breaker 状态。
func (s *PostgresStore) UpsertCircuitState(state PersistentCircuitState) error {
	ctx := context.Background()
	var openedAt any
	if state.CircuitOpenedAt != nil {
		openedAt = state.CircuitOpenedAt.Unix()
	}
	var halfOpenAt any
	if state.HalfOpenAt != nil {
		halfOpenAt = state.HalfOpenAt.Unix()
	}
	var nextRetryAt any
	if state.NextRetryAt != nil {
		nextRetryAt = state.NextRetryAt.Unix()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO circuit_states (
			metrics_key, api_type, base_url, key_mask, circuit_state,
			circuit_opened_at, half_open_at, next_retry_at,
			backoff_level, half_open_successes, consecutive_failures, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT(metrics_key, api_type) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			key_mask = EXCLUDED.key_mask,
			circuit_state = EXCLUDED.circuit_state,
			circuit_opened_at = EXCLUDED.circuit_opened_at,
			half_open_at = EXCLUDED.half_open_at,
			next_retry_at = EXCLUDED.next_retry_at,
			backoff_level = EXCLUDED.backoff_level,
			half_open_successes = EXCLUDED.half_open_successes,
			consecutive_failures = EXCLUDED.consecutive_failures,
			updated_at = EXCLUDED.updated_at
	`, state.MetricsKey, state.APIType, state.BaseURL, state.KeyMask, state.CircuitState, openedAt, halfOpenAt, nextRetryAt, state.BackoffLevel, state.HalfOpenSuccesses, state.ConsecutiveFailures, time.Now().Unix())
	return err
}

// LoadLatestTimestamps 从全量历史记录中查询每个 key 的最后成功/失败时间
func (s *PostgresStore) LoadLatestTimestamps(apiType string) (map[string]*KeyLatestTimestamps, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `
		SELECT
			metrics_key,
			base_url,
			key_mask,
			MAX(CASE WHEN success = TRUE THEN timestamp END) AS last_success,
			MAX(CASE WHEN success = FALSE THEN timestamp END) AS last_failure
		FROM request_records
		WHERE api_type = $1
		GROUP BY metrics_key, base_url, key_mask
	`, apiType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*KeyLatestTimestamps)
	for rows.Next() {
		var metricsKey, baseURL, keyMask string
		var lastSuccessTS, lastFailureTS *int64

		if err := rows.Scan(&metricsKey, &baseURL, &keyMask, &lastSuccessTS, &lastFailureTS); err != nil {
			return nil, err
		}

		kt := &KeyLatestTimestamps{
			BaseURL: baseURL,
			KeyMask: keyMask,
		}
		if lastSuccessTS != nil {
			t := time.Unix(*lastSuccessTS, 0)
			kt.LastSuccessAt = &t
		}
		if lastFailureTS != nil {
			t := time.Unix(*lastFailureTS, 0)
			kt.LastFailureAt = &t
		}
		result[metricsKey] = kt
	}

	return result, rows.Err()
}

// CleanupOldRecords 清理过期数据
func (s *PostgresStore) CleanupOldRecords(before time.Time) (int64, error) {
	ctx := context.Background()
	result, err := s.pool.Exec(ctx,
		"DELETE FROM request_records WHERE timestamp < $1",
		before.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteRecordsByMetricsKeys 按 metrics_key 和 api_type 批量删除记录
func (s *PostgresStore) DeleteRecordsByMetricsKeys(metricsKeys []string, apiType string) (int64, error) {
	if len(metricsKeys) == 0 {
		return 0, nil
	}

	// 获取 flush 锁，确保删除期间不会有后台 flush 写入新记录
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	// 先刷新缓冲区，确保待删除的记录已写入数据库
	s.flush()

	// 分批删除，避免单次 IN 子句过大
	const batchSize = 500
	var totalDeleted int64

	ctx := context.Background()
	for i := 0; i < len(metricsKeys); i += batchSize {
		end := i + batchSize
		if end > len(metricsKeys) {
			end = len(metricsKeys)
		}
		batch := metricsKeys[i:end]

		// 构建 IN 子句的占位符
		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)+1)
		args = append(args, apiType) // 第一个参数是 api_type
		for j := range batch {
			placeholders[j] = fmt.Sprintf("$%d", j+2) // $2, $3, ... ($1 是 api_type)
		}

		query := fmt.Sprintf(
			"DELETE FROM request_records WHERE api_type = $1 AND metrics_key IN (%s)",
			strings.Join(placeholders, ","),
		)

		result, err := s.pool.Exec(ctx, query, args...)
		if err != nil {
			return totalDeleted, fmt.Errorf("batch %d-%d failed: %w", i, end, err)
		}
		totalDeleted += result.RowsAffected()
	}

	return totalDeleted, nil
}

// DeleteCircuitStatesByMetricsKeys 按 metrics_key 和 api_type 批量删除 breaker 状态。
func (s *PostgresStore) DeleteCircuitStatesByMetricsKeys(metricsKeys []string, apiType string) (int64, error) {
	if len(metricsKeys) == 0 {
		return 0, nil
	}

	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	const batchSize = 500
	var totalDeleted int64

	ctx := context.Background()
	for i := 0; i < len(metricsKeys); i += batchSize {
		end := i + batchSize
		if end > len(metricsKeys) {
			end = len(metricsKeys)
		}
		batch := metricsKeys[i:end]
		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)+1)
		args = append(args, apiType)
		for j := range batch {
			placeholders[j] = fmt.Sprintf("$%d", j+2)
		}
		query := fmt.Sprintf(
			"DELETE FROM circuit_states WHERE api_type = $1 AND metrics_key IN (%s)",
			strings.Join(placeholders, ","),
		)
		result, err := s.pool.Exec(ctx, query, args...)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete circuit states batch %d-%d failed: %w", i, end, err)
		}
		totalDeleted += result.RowsAffected()
	}

	return totalDeleted, nil
}

// flushLoop 定时刷新循环
func (s *PostgresStore) flushLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 获取 flush 锁，与 DeleteRecordsByMetricsKeys 串行化
			s.flushMu.Lock()
			s.flush()
			s.flushMu.Unlock()
		case <-s.stopCh:
			// 关闭前最后一次刷新
			s.flushMu.Lock()
			s.flush()
			s.flushMu.Unlock()
			return
		}
	}
}

// cleanupLoop 定期清理循环
func (s *PostgresStore) cleanupLoop() {
	defer s.wg.Done()

	// 启动时先清理一次
	s.doCleanup()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.doCleanup()
		case <-s.stopCh:
			return
		}
	}
}

// doCleanup 执行清理
func (s *PostgresStore) doCleanup() {
	cutoff := time.Now().AddDate(0, 0, -s.retentionDays)
	deleted, err := s.CleanupOldRecords(cutoff)
	if err != nil {
		log.Printf("[Postgres-Cleanup] 警告: 清理过期指标记录失败: %v", err)
	} else if deleted > 0 {
		log.Printf("[Postgres-Cleanup] 已清理 %d 条过期指标记录（超过 %d 天）", deleted, s.retentionDays)
	}
}

// Close 关闭存储
func (s *PostgresStore) Close() error {
	// 标记为已关闭，阻止新记录
	s.bufferMu.Lock()
	s.closed = true
	s.bufferMu.Unlock()

	// 停止后台循环（flushLoop 会在退出前执行最后一次 flush）
	close(s.stopCh)
	s.wg.Wait()

	// 等待所有 AddRecord 触发的异步 flush goroutine 完成
	s.asyncFlushWg.Wait()

	s.pool.Close()
	return nil
}

// GetRecordCount 获取记录总数（用于调试）
func (s *PostgresStore) GetRecordCount() (int64, error) {
	ctx := context.Background()
	var count int64
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM request_records").Scan(&count)
	return count, err
}

// QueryAggregatedHistory 从 Postgres 查询聚合历史数据
// 按指定时间间隔聚合，可选按 apiType、metricsKey、baseURL 过滤
func (s *PostgresStore) QueryAggregatedHistory(apiType string, since time.Time, intervalSeconds int64, metricsKey string, baseURL string) ([]AggregatedBucket, error) {
	// 先等待在途 flush 完成，再刷新当前缓冲区，确保查询视图尽可能完整。
	s.flushMu.Lock()
	s.flushBufferLocked()
	s.flushMu.Unlock()

	ctx := context.Background()
	query := `
		SELECT
			(timestamp / $1) * $1 AS bucket,
			COUNT(*) AS total,
			SUM(CASE WHEN success = TRUE THEN 1 ELSE 0 END) AS success_count,
			SUM(input_tokens) AS input_tokens,
			SUM(output_tokens) AS output_tokens,
			SUM(cache_creation_tokens) AS cache_creation_tokens,
			SUM(cache_read_tokens) AS cache_read_tokens
		FROM request_records
		WHERE api_type = $2 AND timestamp >= $3`

	args := []any{intervalSeconds, apiType, since.Unix()}
	argIdx := 4

	if metricsKey != "" {
		query += fmt.Sprintf(" AND metrics_key = $%d", argIdx)
		args = append(args, metricsKey)
		argIdx++
	}
	if baseURL != "" {
		query += fmt.Sprintf(" AND base_url = $%d", argIdx)
		args = append(args, baseURL)
		argIdx++
	}

	query += " GROUP BY bucket ORDER BY bucket"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询聚合历史失败: %w", err)
	}
	defer rows.Close()

	var results []AggregatedBucket
	for rows.Next() {
		var bucket int64
		var b AggregatedBucket
		if err := rows.Scan(&bucket, &b.TotalRequests, &b.SuccessCount, &b.InputTokens, &b.OutputTokens, &b.CacheCreationTokens, &b.CacheReadTokens); err != nil {
			return nil, fmt.Errorf("扫描聚合结果失败: %w", err)
		}
		b.Timestamp = time.Unix(bucket, 0)
		results = append(results, b)
	}
	return results, rows.Err()
}

// flushBuffer 手动刷新写入缓冲区（查询前调用，确保数据完整性）
func (s *PostgresStore) flushBuffer() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	s.flushBufferLocked()
}

// flushBufferLocked 在调用方已持有 flushMu 时刷新写入缓冲区
func (s *PostgresStore) flushBufferLocked() {
	s.bufferMu.Lock()
	records := make([]PersistentRecord, len(s.writeBuffer))
	copy(records, s.writeBuffer)
	s.writeBuffer = s.writeBuffer[:0]
	s.bufferMu.Unlock()

	if len(records) > 0 {
		if err := s.batchInsertRecords(records); err != nil {
			log.Printf("[Postgres-Flush] 手动刷新失败: %v", err)
			s.requeueRecords(records, "[Postgres-Flush]")
		}
	}
}

func (s *PostgresStore) requeueRecords(records []PersistentRecord, logPrefix string) {
	if len(records) == 0 {
		return
	}

	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()

	if len(s.writeBuffer) < s.batchSize*10 {
		s.writeBuffer = append(records, s.writeBuffer...)
		return
	}

	log.Printf("%s 警告: 写入缓冲区已满，丢弃 %d 条记录", logPrefix, len(records))
}

// --- 迁移辅助函数 ---

func buildMetricsKeyMigrationMap(cfg config.Config) map[string]map[string]metricsKeyMigrationCandidates {
	mapping := map[string]map[string]metricsKeyMigrationCandidates{
		"messages":  {},
		"responses": {},
		"gemini":    {},
		"chat":      {},
		"images":    {},
	}

	addUpstreamMetricsKeyMappings(mapping["messages"], cfg.Upstream, "claude")
	addUpstreamMetricsKeyMappings(mapping["responses"], cfg.ResponsesUpstream, "responses")
	addUpstreamMetricsKeyMappings(mapping["gemini"], cfg.GeminiUpstream, "gemini")
	addUpstreamMetricsKeyMappings(mapping["chat"], cfg.ChatUpstream, "openai")
	addUpstreamMetricsKeyMappings(mapping["images"], cfg.ImagesUpstream, "openai")

	return mapping
}

func legacyMetricsKeysForMigration(baseURL, apiKey, serviceType string) []string {
	seen := make(map[string]struct{})
	keys := make([]string, 0, 6)
	add := func(rawBaseURL string) {
		if rawBaseURL == "" {
			return
		}
		metricsKey := GenerateMetricsKey(rawBaseURL, apiKey)
		if _, exists := seen[metricsKey]; exists {
			return
		}
		seen[metricsKey] = struct{}{}
		keys = append(keys, metricsKey)
	}

	for _, variant := range utils.EquivalentBaseURLVariants(baseURL, serviceType) {
		add(variant)
	}
	add(utils.MetricsIdentityBaseURL(baseURL, serviceType))
	return keys
}

func addUpstreamMetricsKeyMappings(targets map[string]metricsKeyMigrationCandidates, upstreams []config.UpstreamConfig, defaultServiceType string) {
	for _, upstream := range upstreams {
		serviceType := upstream.ServiceType
		if serviceType == "" {
			serviceType = defaultServiceType
		}
		baseURLs := upstream.GetAllBaseURLs()
		if len(baseURLs) == 0 {
			continue
		}
		keys := deduplicateMetricMigrationKeys(upstream.APIKeys, upstream.HistoricalAPIKeys)
		for _, baseURL := range baseURLs {
			identityBaseURL := utils.MetricsIdentityBaseURL(baseURL, serviceType)
			for _, apiKey := range keys {
				if apiKey == "" {
					continue
				}
				migrationTarget := metricsKeyMigrationTarget{
					MetricsKey: GenerateMetricsIdentityKey(baseURL, apiKey, serviceType),
					BaseURL:    identityBaseURL,
				}
				for _, legacyKey := range legacyMetricsKeysForMigration(baseURL, apiKey, serviceType) {
					candidate, exists := targets[legacyKey]
					if !exists {
						targets[legacyKey] = metricsKeyMigrationCandidates{Primary: migrationTarget}
						continue
					}
					if candidate.Primary == migrationTarget || containsMigrationConflict(candidate.Conflicts, migrationTarget) {
						continue
					}
					candidate.Conflicts = append(candidate.Conflicts, migrationTarget)
					targets[legacyKey] = candidate
				}
			}
		}
	}
}

func containsMigrationConflict(conflicts []metricsKeyMigrationTarget, target metricsKeyMigrationTarget) bool {
	for _, conflict := range conflicts {
		if conflict == target {
			return true
		}
	}
	return false
}

func deduplicateMetricMigrationKeys(groups ...[]string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, group := range groups {
		for _, key := range group {
			if key == "" {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, key)
		}
	}
	return result
}

func mergePersistedCircuitState(dst, src *persistedCircuitStateRow) {
	if dst == nil || src == nil {
		return
	}
	if circuitStateSeverity(src.CircuitState) > circuitStateSeverity(dst.CircuitState) {
		dst.CircuitState = src.CircuitState
	}
	if src.BackoffLevel > dst.BackoffLevel {
		dst.BackoffLevel = src.BackoffLevel
	}
	if src.HalfOpenSuccesses > dst.HalfOpenSuccesses {
		dst.HalfOpenSuccesses = src.HalfOpenSuccesses
	}
	if src.ConsecutiveFailures > dst.ConsecutiveFailures {
		dst.ConsecutiveFailures = src.ConsecutiveFailures
	}
	if src.UpdatedAt > dst.UpdatedAt {
		dst.UpdatedAt = src.UpdatedAt
		if src.CircuitOpenedAt != nil {
			dst.CircuitOpenedAt = src.CircuitOpenedAt
		}
		if src.HalfOpenAt != nil {
			dst.HalfOpenAt = src.HalfOpenAt
		}
		if src.NextRetryAt != nil {
			dst.NextRetryAt = src.NextRetryAt
		}
	}
}

func circuitStateSeverity(state string) int {
	switch state {
	case "open":
		return 3
	case "half_open":
		return 2
	default:
		return 1
	}
}
