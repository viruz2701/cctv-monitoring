// Package db — детекция медленных запросов и рекомендации по индексам.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-DB: Database Optimization — Slow Query Detector
//
// SlowQueryDetector анализирует pg_stat_statements, выявляет медленные
// запросы и генерирует рекомендации по индексам на основе seq scans,
// nested loops и высокой cardinality.
//
// Соответствие:
//   - ISO 27001 A.12.6.1 (Capacity Management)
//   - IEC 62443-3-3 SR 7.2 (Performance Monitoring)
//   - СТБ 34.101.27 п. 7.3 (Управление ресурсами)
//
// ═══════════════════════════════════════════════════════════════════════════
package db

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Типы ───────────────────────────────────────────────────────────────

// SlowQuery представляет один медленный запрос.
type SlowQuery struct {
	// Идентификатор запроса (pg_stat_statements)
	QueryID uint64 `json:"query_id"`
	Query   string `json:"query"`
	DBName  string `json:"db_name"`

	// Статистика выполнения
	Calls        int64   `json:"calls"`
	TotalTimeMs  float64 `json:"total_time_ms"`
	MeanTimeMs   float64 `json:"mean_time_ms"`
	MaxTimeMs    float64 `json:"max_time_ms"`
	MinTimeMs    float64 `json:"min_time_ms"`
	StddevTimeMs float64 `json:"stddev_time_ms"`

	// Статистика строк
	Rows     int64   `json:"rows"`
	MeanRows float64 `json:"mean_rows"`

	// I/O статистика
	SharedReads  int64 `json:"shared_reads"`
	SharedWrites int64 `json:"shared_writes"`
	LocalReads   int64 `json:"local_reads"`
	LocalWrites  int64 `json:"local_writes"`

	// План запроса (EXPLAIN ANALYZE)
	QueryPlan string `json:"query_plan,omitempty"`

	// Временные метки
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	DetectedAt time.Time `json:"detected_at"`
}

// IndexRecommendation представляет рекомендацию по созданию индекса.
type IndexRecommendation struct {
	// Таблица и колонки
	Table     string   `json:"table"`
	Columns   []string `json:"columns"`
	IndexType string   `json:"index_type"` // btree, hash, gin, gist, brin

	// Тип проблемы
	IssueType string `json:"issue_type"` // seq_scan, nested_loop, high_cardinality

	// SQL для создания индекса
	CreateIndexSQL string `json:"create_index_sql"`

	// Оценка эффекта
	EstimatedImprovement string  `json:"estimated_improvement"`
	Confidence           float64 `json:"confidence"` // 0.0 — 1.0

	// Контекст
	QueryPattern  string `json:"query_pattern,omitempty"`
	TableRowCount int64  `json:"table_row_count,omitempty"`

	// Временные метки
	DetectedAt time.Time `json:"detected_at"`

	// Статус (pending, applied, rejected)
	Status string `json:"status"`
}

// SlowQueryDetector реализует детекцию медленных запросов и рекомендации.
type SlowQueryDetector struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	cfg    SlowQueryDetectorConfig

	mu              sync.RWMutex
	slowQueries     []*SlowQuery
	recommendations []*IndexRecommendation

	// Кэш схемы БД для анализа запросов
	schemaCache map[string]*tableSchema
	schemaMu    sync.RWMutex
}

// SlowQueryDetectorConfig — конфигурация детектора.
type SlowQueryDetectorConfig struct {
	// SlowQueryThresholdMs — порог медленного запроса (mean time).
	SlowQueryThresholdMs float64

	// MinCalls — минимальное количество вызовов для анализа.
	MinCalls int64

	// CollectInterval — периодичность сбора статистики.
	CollectInterval time.Duration

	// MaxSlowQueries — максимальное количество хранимых медленных запросов.
	MaxSlowQueries int

	// MaxRecommendations — максимальное количество рекомендаций.
	MaxRecommendations int

	// ExplainThresholdMs — порог для запуска EXPLAIN ANALYZE.
	ExplainThresholdMs float64

	// EnableIndexRecommendations — включить рекомендации индексов.
	EnableIndexRecommendations bool

	// MinTableRowCount — минимальное количество строк для рекомендации индекса.
	MinTableRowCount int64
}

// DefaultSlowQueryDetectorConfig возвращает конфигурацию по умолчанию.
func DefaultSlowQueryDetectorConfig() SlowQueryDetectorConfig {
	return SlowQueryDetectorConfig{
		SlowQueryThresholdMs:       100, // 100ms
		MinCalls:                   10,
		CollectInterval:            60 * time.Second,
		MaxSlowQueries:             100,
		MaxRecommendations:         50,
		ExplainThresholdMs:         1000, // 1s
		EnableIndexRecommendations: true,
		MinTableRowCount:           10000,
	}
}

// tableSchema представляет схему таблицы для анализа.
type tableSchema struct {
	Name     string
	Columns  []columnInfo
	RowCount int64
}

type columnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Indexed  bool
}

// ── Конструктор ────────────────────────────────────────────────────────

// NewSlowQueryDetector создаёт новый SlowQueryDetector.
func NewSlowQueryDetector(pool *pgxpool.Pool, cfg SlowQueryDetectorConfig, logger *slog.Logger) *SlowQueryDetector {
	d := &SlowQueryDetector{
		pool:            pool,
		logger:          logger.With("component", "db.slow_query_detector"),
		cfg:             cfg,
		slowQueries:     make([]*SlowQuery, 0, cfg.MaxSlowQueries),
		recommendations: make([]*IndexRecommendation, 0, cfg.MaxRecommendations),
		schemaCache:     make(map[string]*tableSchema),
	}

	// Инициализация кэша схемы
	d.refreshSchemaCache()

	// Запускаем сбор
	go d.collectLoop()

	logger.Info("slow query detector initialized",
		"threshold_ms", cfg.SlowQueryThresholdMs,
		"min_calls", cfg.MinCalls,
		"interval", cfg.CollectInterval,
	)
	return d
}

// ── Основные методы ─────────────────────────────────────────────────────

// GetSlowQueries возвращает список медленных запросов.
func (d *SlowQueryDetector) GetSlowQueries() []*SlowQuery {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*SlowQuery, len(d.slowQueries))
	copy(result, d.slowQueries)
	return result
}

// GetIndexRecommendations возвращает список рекомендаций по индексам.
func (d *SlowQueryDetector) GetIndexRecommendations() []*IndexRecommendation {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*IndexRecommendation, len(d.recommendations))
	copy(result, d.recommendations)
	return result
}

// ── Внутренние методы ──────────────────────────────────────────────────

// collectLoop периодически собирает статистику.
func (d *SlowQueryDetector) collectLoop() {
	ticker := time.NewTicker(d.cfg.CollectInterval)
	defer ticker.Stop()

	// Первый сбор сразу
	d.collectSlowQueries()

	for range ticker.C {
		d.collectSlowQueries()
	}
}

// collectSlowQueries собирает медленные запросы из pg_stat_statements.
func (d *SlowQueryDetector) collectSlowQueries() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Проверяем наличие pg_stat_statements
	var hasExtension bool
	err := d.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements')`,
	).Scan(&hasExtension)

	if err != nil || !hasExtension {
		d.logger.Warn("pg_stat_statements extension not available, slow query detection limited",
			"error", err,
		)
		// Fallback: собираем из pg_stat_activity
		d.collectFromActivity(ctx)
		return
	}

	// Собираем медленные запросы из pg_stat_statements
	rows, err := d.pool.Query(ctx, `
		SELECT
			queryid,
			left(query, 1000) as query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			min_exec_time,
			stddev_exec_time,
			rows,
			shared_blks_read,
			shared_blks_written,
			local_blks_read,
			local_blks_written,
			now() - pg_postmaster_start_time() + '1 second' as first_seen,
			now() as last_seen
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
			AND calls >= $2
			AND query NOT LIKE '%pg_stat%'
			AND query NOT LIKE '%pg_catalog%'
			AND query NOT LIKE '%information_schema%'
		ORDER BY mean_exec_time DESC
		LIMIT $3
	`, d.cfg.SlowQueryThresholdMs, d.cfg.MinCalls, d.cfg.MaxSlowQueries)
	if err != nil {
		d.logger.Error("failed to query pg_stat_statements", "error", err)
		return
	}
	defer rows.Close()

	var queries []*SlowQuery
	for rows.Next() {
		sq := &SlowQuery{DetectedAt: time.Now().UTC()}
		var firstSeenDur string
		if err := rows.Scan(
			&sq.QueryID,
			&sq.Query,
			&sq.Calls,
			&sq.TotalTimeMs,
			&sq.MeanTimeMs,
			&sq.MaxTimeMs,
			&sq.MinTimeMs,
			&sq.StddevTimeMs,
			&sq.Rows,
			&sq.SharedReads,
			&sq.SharedWrites,
			&sq.LocalReads,
			&sq.LocalWrites,
			&firstSeenDur,
			&sq.LastSeen,
		); err != nil {
			d.logger.Error("failed to scan slow query row", "error", err)
			continue
		}

		// Определяем имя БД
		_ = d.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&sq.DBName)

		// Для запросов > ExplainThresholdMs получаем план
		if sq.MeanTimeMs >= d.cfg.ExplainThresholdMs {
			plan, err := d.getQueryPlan(ctx, sq.Query)
			if err == nil {
				sq.QueryPlan = plan
			}
		}

		queries = append(queries, sq)
	}

	d.mu.Lock()
	d.slowQueries = queries
	d.mu.Unlock()

	d.logger.Debug("slow queries collected", "count", len(queries))

	// Генерируем рекомендации по индексам
	if d.cfg.EnableIndexRecommendations && len(queries) > 0 {
		d.generateRecommendations(ctx, queries)
	}
}

// collectFromActivity собирает медленные запросы из pg_stat_activity (fallback).
func (d *SlowQueryDetector) collectFromActivity(ctx context.Context) {
	rows, err := d.pool.Query(ctx, `
		SELECT
			pid,
			coalesce(left(query, 500), ''),
			state,
			now() - query_start as duration,
			wait_event_type,
			wait_event
		FROM pg_stat_activity
		WHERE state = 'active'
			AND query NOT LIKE '%pg_stat%'
			AND query NOT LIKE '%pg_catalog%'
			AND now() - query_start > interval '100 milliseconds'
		ORDER BY duration DESC
		LIMIT $1
	`, d.cfg.MaxSlowQueries)
	if err != nil {
		d.logger.Error("failed to query pg_stat_activity", "error", err)
		return
	}
	defer rows.Close()

	var queries []*SlowQuery
	for rows.Next() {
		var pid int32
		var query, state, waitEventType, waitEvent string
		var duration time.Duration

		if err := rows.Scan(&pid, &query, &state, &duration, &waitEventType, &waitEvent); err != nil {
			continue
		}

		sq := &SlowQuery{
			Query:      query,
			MeanTimeMs: duration.Seconds() * 1000,
			Calls:      1,
			DetectedAt: time.Now().UTC(),
			LastSeen:   time.Now(),
		}
		_ = d.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&sq.DBName)
		queries = append(queries, sq)
	}

	d.mu.Lock()
	d.slowQueries = queries
	d.mu.Unlock()
}

// getQueryPlan выполняет EXPLAIN ANALYZE для запроса (в read-only транзакции).
func (d *SlowQueryDetector) getQueryPlan(ctx context.Context, query string) (string, error) {
	// Ограничиваем: только SELECT запросы
	trimmed := strings.TrimSpace(query)
	if !strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") {
		return "", fmt.Errorf("only SELECT queries can be explained")
	}

	// Оборачиваем в EXPLAIN ANALYZE
	explainSQL := fmt.Sprintf("EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT TEXT) %s", trimmed)

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Устанавливаем read-only для безопасности
	if _, err := tx.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return "", fmt.Errorf("set transaction read only: %w", err)
	}

	rows, err := tx.Query(ctx, explainSQL)
	if err != nil {
		return "", fmt.Errorf("explain query: %w", err)
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		planLines = append(planLines, line)
	}

	return strings.Join(planLines, "\n"), nil
}

// generateRecommendations анализирует медленные запросы и генерирует рекомендации.
func (d *SlowQueryDetector) generateRecommendations(ctx context.Context, queries []*SlowQuery) {
	var recommendations []*IndexRecommendation

	for _, sq := range queries {
		if sq.QueryPlan == "" && sq.MeanTimeMs < d.cfg.ExplainThresholdMs {
			continue
		}

		// Анализируем запрос через pg_hint_plan или встроенный парсер
		recs := d.analyzeQueryForIndexes(ctx, sq)
		recommendations = append(recommendations, recs...)
	}

	// Дедуплицируем и сортируем по confidence
	recommendations = d.deduplicateRecommendations(recommendations)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Confidence > recommendations[j].Confidence
	})

	// Ограничиваем количество
	if len(recommendations) > d.cfg.MaxRecommendations {
		recommendations = recommendations[:d.cfg.MaxRecommendations]
	}

	d.mu.Lock()
	d.recommendations = recommendations
	d.mu.Unlock()

	d.logger.Debug("index recommendations generated", "count", len(recommendations))
}

// analyzeQueryForIndexes анализирует запрос на предмет отсутствующих индексов.
func (d *SlowQueryDetector) analyzeQueryForIndexes(ctx context.Context, sq *SlowQuery) []*IndexRecommendation {
	var recommendations []*IndexRecommendation

	// Простой эвристический анализ:
	// 1. Ищем WHERE conditions
	// 2. Ищем JOIN conditions
	// 3. Ищем ORDER BY на неиндексированных колонках
	// 4. Анализируем seq scans из query plan

	queryUpper := strings.ToUpper(sq.Query)

	// Пропускаем DDL и DML кроме SELECT
	if !strings.HasPrefix(strings.TrimSpace(queryUpper), "SELECT") {
		return nil
	}

	// Извлекаем имя таблицы (простой парсинг)
	tableName := d.extractTableName(sq.Query)
	if tableName == "" {
		return nil
	}

	// Получаем схему таблицы
	tableSchema := d.getTableSchema(ctx, tableName)
	if tableSchema == nil {
		return nil
	}

	// Пропускаем маленькие таблицы
	if tableSchema.RowCount < d.cfg.MinTableRowCount {
		return nil
	}

	// Анализируем WHERE условия
	whereConditions := d.extractWhereColumns(sq.Query)
	for _, col := range whereConditions {
		if !d.isColumnIndexed(tableSchema, col) {
			rec := &IndexRecommendation{
				Table:                tableName,
				Columns:              []string{col},
				IndexType:            "btree",
				IssueType:            "seq_scan",
				QueryPattern:         sq.Query,
				TableRowCount:        tableSchema.RowCount,
				DetectedAt:           time.Now().UTC(),
				Status:               "pending",
				EstimatedImprovement: "medium",
			}

			// Оценка confidence на основе размера таблицы и mean_time
			rec.Confidence = d.calculateConfidence(tableSchema.RowCount, sq.MeanTimeMs, len(whereConditions))
			rec.CreateIndexSQL = fmt.Sprintf(
				"CREATE INDEX CONCURRENTLY idx_%s_%s ON %s (%s);",
				tableName, col, tableName, col,
			)
			recommendations = append(recommendations, rec)
		}
	}

	// Проверяем seq scan в query plan
	if sq.QueryPlan != "" && strings.Contains(sq.QueryPlan, "Seq Scan") {
		recs := d.analyzeSeqScans(sq.QueryPlan, tableSchema)
		recommendations = append(recommendations, recs...)
	}

	return recommendations
}

// extractTableName извлекает имя таблицы из SQL запроса.
func (d *SlowQueryDetector) extractTableName(query string) string {
	queryUpper := strings.ToUpper(query)

	// Ищем FROM
	fromIdx := strings.Index(queryUpper, "FROM")
	if fromIdx < 0 {
		return ""
	}

	afterFrom := strings.TrimSpace(query[fromIdx+4:])
	// Берем первое слово (имя таблицы), отсекаем псевдонимы и JOIN
	parts := strings.Fields(afterFrom)
	if len(parts) == 0 {
		return ""
	}

	// Убираем схему (public.table -> table)
	tableName := strings.Trim(parts[0], `"`)

	// Если есть схема
	if idx := strings.Index(tableName, "."); idx > 0 {
		tableName = tableName[idx+1:]
	}

	return tableName
}

// extractWhereColumns извлекает колонки из WHERE условий.
func (d *SlowQueryDetector) extractWhereColumns(query string) []string {
	queryUpper := strings.ToUpper(query)

	whereIdx := strings.Index(queryUpper, "WHERE")
	if whereIdx < 0 {
		return nil
	}

	// Берем текст после WHERE
	afterWhere := query[whereIdx+5:]

	// Ищем конец WHERE (ORDER BY, GROUP BY, LIMIT, HAVING)
	endKeywords := []string{"ORDER BY", "GROUP BY", "LIMIT", "HAVING", "RETURNING"}
	whereClause := afterWhere
	for _, kw := range endKeywords {
		if idx := strings.Index(strings.ToUpper(whereClause), kw); idx > 0 {
			whereClause = whereClause[:idx]
		}
	}

	// Парсим колонки из условий вида: column = value, column > value, column IN (...)
	var columns []string
	parts := strings.Fields(whereClause)
	for i, part := range parts {
		// Ищем операторы сравнения
		if i > 0 && isComparisonOperator(part) && i-1 < len(parts) {
			col := strings.Trim(parts[i-1], `"'`)
			if isLikelyColumn(col) {
				columns = append(columns, col)
			}
		}
		// Ищем IN, BETWEEN, LIKE
		if strings.ToUpper(part) == "IN" && i > 0 {
			col := strings.Trim(parts[i-1], `"'`)
			if isLikelyColumn(col) {
				columns = append(columns, col)
			}
		}
	}

	return uniqueStrings(columns)
}

// analyzeSeqScans анализирует Seq Scan из query plan.
func (d *SlowQueryDetector) analyzeSeqScans(plan string, schema *tableSchema) []*IndexRecommendation {
	var recommendations []*IndexRecommendation

	lines := strings.Split(plan, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Seq Scan") {
			// Извлекаем имя таблицы
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, "on") && i+1 < len(parts) {
					tableName := strings.Trim(parts[i+1], `"'`)
					// Убираем псевдонимы
					if idx := strings.Index(tableName, "."); idx > 0 {
						tableName = tableName[idx+1:]
					}

					// Ищем фильтр
					if strings.Contains(line, "Filter:") {
						filterIdx := strings.Index(line, "Filter:")
						filterClause := line[filterIdx+7:]
						colName := extractColumnFromFilter(filterClause)
						if colName != "" && !d.isColumnIndexed(schema, colName) {
							rec := &IndexRecommendation{
								Table:                tableName,
								Columns:              []string{colName},
								IndexType:            "btree",
								IssueType:            "seq_scan",
								QueryPattern:         line,
								TableRowCount:        schema.RowCount,
								DetectedAt:           time.Now().UTC(),
								Status:               "pending",
								EstimatedImprovement: "high",
								Confidence:           0.8,
								CreateIndexSQL: fmt.Sprintf(
									"CREATE INDEX CONCURRENTLY idx_%s_%s ON %s (%s);",
									tableName, colName, tableName, colName,
								),
							}
							recommendations = append(recommendations, rec)
						}
					}
				}
			}
		}
		// Анализируем Nested Loop
		if strings.Contains(line, "Nested Loop") {
			// Nested Loop часто означает отсутствие индекса для JOIN
		}
	}

	return recommendations
}

// ── Вспомогательные методы ─────────────────────────────────────────────

// refreshSchemaCache обновляет кэш схемы БД.
func (d *SlowQueryDetector) refreshSchemaCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx, `
		SELECT
			t.relname as table_name,
			n.nspname as schema_name,
			COALESCE(s.n_live_tup, 0) as row_count
		FROM pg_class t
		JOIN pg_namespace n ON t.relnamespace = n.oid
		LEFT JOIN pg_stat_user_tables s ON t.oid = s.relid
		WHERE t.relkind = 'r'
			AND n.nspname NOT IN ('pg_catalog', 'information_schema')
			AND n.nspname NOT LIKE 'pg_toast%'
		ORDER BY s.n_live_tup DESC NULLS LAST
	`)
	if err != nil {
		d.logger.Error("failed to refresh schema cache", "error", err)
		return
	}
	defer rows.Close()

	d.schemaMu.Lock()
	d.schemaCache = make(map[string]*tableSchema)
	for rows.Next() {
		var tableName, schemaName string
		var rowCount int64
		if err := rows.Scan(&tableName, &schemaName, &rowCount); err != nil {
			continue
		}
		fullName := fmt.Sprintf("%s.%s", schemaName, tableName)
		d.schemaCache[tableName] = &tableSchema{
			Name:     fullName,
			RowCount: rowCount,
		}
		d.schemaCache[fullName] = d.schemaCache[tableName]
	}
	d.schemaMu.Unlock()
}

// getTableSchema возвращает схему таблицы с колонками.
func (d *SlowQueryDetector) getTableSchema(ctx context.Context, tableName string) *tableSchema {
	d.schemaMu.RLock()
	schema, ok := d.schemaCache[tableName]
	d.schemaMu.RUnlock()

	if ok && len(schema.Columns) > 0 {
		return schema
	}

	// Загружаем колонки
	rows, err := d.pool.Query(ctx, `
		SELECT
			a.attname as column_name,
			t.typname as data_type,
			a.attnotnull as not_null,
			COALESCE(ix.indexed, false) as indexed
		FROM pg_class c
		JOIN pg_attribute a ON a.attrelid = c.oid
		JOIN pg_type t ON a.atttypid = t.oid
		LEFT JOIN (
			SELECT indrelid, unnest(indkey) as attnum
			FROM pg_index
		) ix ON ix.indrelid = c.oid AND ix.attnum = a.attnum
		WHERE c.relname = $1
			AND a.attnum > 0
			AND NOT a.attisdropped
		ORDER BY a.attnum
	`, tableName)
	if err != nil {
		return schema
	}
	defer rows.Close()

	var columns []columnInfo
	for rows.Next() {
		var col columnInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Indexed); err != nil {
			continue
		}
		col.Nullable = !col.Nullable // attnotnull -> nullable
		columns = append(columns, col)
	}

	if schema == nil {
		schema = &tableSchema{Name: tableName}
	}
	schema.Columns = columns

	d.schemaMu.Lock()
	d.schemaCache[tableName] = schema
	d.schemaMu.Unlock()

	return schema
}

// isColumnIndexed проверяет, индексирована ли колонка.
func (d *SlowQueryDetector) isColumnIndexed(schema *tableSchema, column string) bool {
	if schema == nil {
		return false
	}
	for _, col := range schema.Columns {
		if col.Name == column {
			return col.Indexed
		}
	}
	return false
}

// calculateConfidence вычисляет confidence для рекомендации.
func (d *SlowQueryDetector) calculateConfidence(rowCount int64, meanTimeMs float64, conditionCount int) float64 {
	// Факторы:
	// - Размер таблицы (>100k = high confidence)
	// - Mean time (>500ms = high confidence)
	// - Количество условий (>1 = higher confidence)
	confidence := 0.5

	if rowCount > 100000 {
		confidence += 0.2
	} else if rowCount > 50000 {
		confidence += 0.1
	}

	if meanTimeMs > 500 {
		confidence += 0.2
	} else if meanTimeMs > 200 {
		confidence += 0.1
	}

	if conditionCount > 1 {
		confidence += 0.1
	}

	return math.Min(confidence, 1.0)
}

// deduplicateRecommendations удаляет дубликаты рекомендаций.
func (d *SlowQueryDetector) deduplicateRecommendations(recs []*IndexRecommendation) []*IndexRecommendation {
	seen := make(map[string]bool)
	var result []*IndexRecommendation

	for _, rec := range recs {
		key := fmt.Sprintf("%s:%s", rec.Table, strings.Join(rec.Columns, ","))
		if !seen[key] {
			seen[key] = true
			result = append(result, rec)
		}
	}

	return result
}

// ── Утилиты ─────────────────────────────────────────────────────────────

// isComparisonOperator проверяет, является ли строка оператором сравнения.
func isComparisonOperator(s string) bool {
	ops := []string{"=", "<", ">", "<=", ">=", "<>", "!=", "LIKE", "ILIKE"}
	upper := strings.ToUpper(s)
	for _, op := range ops {
		if upper == op || upper == strings.ToUpper(op) {
			return true
		}
	}
	return false
}

// isLikelyColumn проверяет, похожа ли строка на имя колонки.
func isLikelyColumn(s string) bool {
	if s == "" {
		return false
	}
	// Исключаем числа, строки, ключевые слова
	upper := strings.ToUpper(s)
	if isComparisonOperator(s) || upper == "AND" || upper == "OR" || upper == "NOT" ||
		upper == "NULL" || upper == "TRUE" || upper == "FALSE" {
		return false
	}
	// Должна содержать буквы
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			return true
		}
	}
	return false
}

// extractColumnFromFilter извлекает имя колонки из Filter условия плана.
func extractColumnFromFilter(filter string) string {
	// Пример: "Filter: (col_name = 'value')"
	filter = strings.TrimSpace(filter)
	filter = strings.TrimPrefix(filter, "(")
	filter = strings.TrimSuffix(filter, ")")

	// Берем первое слово до оператора
	parts := strings.Fields(filter)
	if len(parts) > 0 {
		col := strings.Trim(parts[0], `"'`)
		if isLikelyColumn(col) {
			return col
		}
	}

	return ""
}

// uniqueStrings возвращает уникальные строки из слайса.
func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
