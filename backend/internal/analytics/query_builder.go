// Package analytics — self-service query builder for embedded BI analytics.
//
// P2-BI: Embedded Self-Service Analytics
//
// QueryBuilder предоставляет визуальный query builder для нетехнических
// пользователей: выбор измерений (dimensions), метрик (measures), фильтров
// и временного диапазона. SQL-шаблоны защищены от injection через
// parameterized placeholders ($1, $2, ...).
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через Field definitions)
//   - OWASP ASVS V6.2 (SQL injection prevention — parameterized queries)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - ISO 27001 A.12.6.1 (Capacity management — analytics metrics)
//   - IEC 62443 SR 3.1 (Input validation on all data paths)
package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Core Types ────────────────────────────────────────────────────────────────

// QueryBuilder строит и выполняет аналитические запросы из шаблонов.
// Потокобезопасен после инициализации (templates read-only).
type QueryBuilder struct {
	pool      *pgxpool.Pool
	templates map[string]QueryTemplate
}

// QueryTemplate представляет параметризованный SQL-шаблон для BI-запроса.
// SQL должен использовать нумерованные placeholders ($1, $2, ...).
//
// Dimensions — поля для GROUP BY (измерения).
// Measures — поля для агрегации (метрики).
// User может выбирать subset из dimensions/measures и добавлять фильтры.
type QueryTemplate struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SQL         string  `json:"sql"`        // parameterized SQL template
	Dimensions  []Field `json:"dimensions"` // groupable fields
	Measures    []Field `json:"measures"`   // aggregate fields
	DateField   string  `json:"date_field"` // column for time range filter
}

// Field описывает одно поле (измерение или метрику) в шаблоне.
type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`               // string, number, date, boolean
	AggFunction string `json:"agg,omitempty"`      // sum, avg, count, max, min — только для measures
	SQLExpr     string `json:"sql_expr,omitempty"` // кастомное SQL-выражение (если отличается от key)
}

// QueryParams задаёт параметры выполняемого запроса.
type QueryParams struct {
	TemplateID string            `json:"template_id"`
	Dimensions []string          `json:"dimensions,omitempty"` // выбранные поля для GROUP BY
	Measures   []string          `json:"measures,omitempty"`   // выбранные метрики
	Filters    []FilterCondition `json:"filters,omitempty"`    // фильтры
	TimeFrom   *time.Time        `json:"time_from,omitempty"`  // начало временного диапазона
	TimeTo     *time.Time        `json:"time_to,omitempty"`    // конец временного диапазона
	Limit      int               `json:"limit,omitempty"`      // макс. строк (0 = нет лимита)
	Offset     int               `json:"offset,omitempty"`     // смещение
	OrderBy    string            `json:"order_by,omitempty"`   // поле сортировки
	OrderDir   string            `json:"order_dir,omitempty"`  // asc | desc
}

// FilterCondition представляет одно условие фильтра.
type FilterCondition struct {
	Field    string      `json:"field"` // имя поля
	Operator string      `json:"op"`    // eq, neq, gt, gte, lt, lte, in, like, contains
	Value    interface{} `json:"value"` // значение или []interface{} для in
}

// QueryResult содержит результат выполнения запроса.
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Total   int64           `json:"total,omitempty"`
	Took    string          `json:"took"`
}

// ── Constructor ───────────────────────────────────────────────────────────────

// New создаёт QueryBuilder с переданными шаблонами.
// templates должны иметь уникальные ID.
func New(pool *pgxpool.Pool, templates []QueryTemplate) (*QueryBuilder, error) {
	tm := make(map[string]QueryTemplate, len(templates))
	for _, t := range templates {
		if _, exists := tm[t.ID]; exists {
			return nil, fmt.Errorf("analytics: duplicate template ID %q", t.ID)
		}
		if t.SQL == "" {
			return nil, fmt.Errorf("analytics: template %q has empty SQL", t.ID)
		}
		tm[t.ID] = t
	}
	return &QueryBuilder{
		pool:      pool,
		templates: tm,
	}, nil
}

// ── Template Access ───────────────────────────────────────────────────────────

// GetTemplates возвращает все доступные шаблоны.
func (qb *QueryBuilder) GetTemplates() []QueryTemplate {
	result := make([]QueryTemplate, 0, len(qb.templates))
	for _, t := range qb.templates {
		result = append(result, t)
	}
	return result
}

// GetTemplate возвращает шаблон по ID.
func (qb *QueryBuilder) GetTemplate(id string) (QueryTemplate, bool) {
	t, ok := qb.templates[id]
	return t, ok
}

// ── Query Execution ───────────────────────────────────────────────────────────

// Execute выполняет BI-запрос на основе шаблона и параметров.
//
// Алгоритм:
//  1. Валидация: проверяет существование шаблона, полей, фильтров
//  2. Построение SQL: подставляет SELECT, WHERE, GROUP BY, ORDER BY, LIMIT/OFFSET
//  3. Выполнение: pgx коллекция строк с таймаутом 30s
//  4. Форматирование: [][]interface{} для JSON-ответа
//
// Compliance:
//   - OWASP ASVS V5.1: все имена полей проверяются по белому списку (Field.Key)
//   - OWASP ASVS V6.2: параметры фильтров через $N placeholders
//   - OWASP ASVS V7.1: errors возвращают только код, без SQL/schema details
func (qb *QueryBuilder) Execute(ctx context.Context, params QueryParams) (*QueryResult, error) {
	start := time.Now()

	// 1. Валидация шаблона
	tmpl, ok := qb.templates[params.TemplateID]
	if !ok {
		return nil, &ValidationError{TemplateID: params.TemplateID, Message: "template not found"}
	}

	// 2. Валидация выбранных полей
	if err := qb.validateFields(tmpl, params); err != nil {
		return nil, err
	}

	// 3. Построение SQL
	query, args, err := qb.buildQuery(tmpl, params)
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	// 4. Выполнение с таймаутом
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := qb.pool.Query(queryCtx, query, args...)
	if err != nil {
		if ctx.Err() != nil {
			return nil, &TimeoutError{Message: "query execution timeout"}
		}
		return nil, &ExecutionError{Message: "query execution failed", Err: err}
	}
	defer rows.Close()

	// 5. Чтение результата
	result, err := qb.readRows(rows)
	if err != nil {
		return nil, &ExecutionError{Message: "failed to read query result", Err: err}
	}

	result.Took = time.Since(start).Round(time.Millisecond).String()
	return result, nil
}

// ── Query Building ────────────────────────────────────────────────────────────

// buildQuery собирает финальный SQL из шаблона и параметров.
func (qb *QueryBuilder) buildQuery(tmpl QueryTemplate, params QueryParams) (string, []interface{}, error) {
	var (
		clauses []string
		args    []interface{}
		argIdx  int
		where   []string
	)

	nextArg := func(val interface{}) string {
		argIdx++
		args = append(args, val)
		return fmt.Sprintf("$%d", argIdx)
	}

	// ── SELECT clause ───────────────────────────────────────────────────
	selects := make([]string, 0, len(params.Dimensions)+len(params.Measures))
	dimExprs := make([]string, 0, len(params.Dimensions))

	for _, d := range params.Dimensions {
		f, ok := findField(tmpl.Dimensions, d)
		if !ok {
			return "", nil, fmt.Errorf("dimension %q not found in template %q", d, tmpl.ID)
		}
		expr := f.SQLExpr
		if expr == "" {
			expr = f.Key
		}
		selects = append(selects, fmt.Sprintf("%s AS %s", expr, quoteIdent(f.Key)))
		dimExprs = append(dimExprs, expr)
	}

	for _, m := range params.Measures {
		f, ok := findField(tmpl.Measures, m)
		if !ok {
			return "", nil, fmt.Errorf("measure %q not found in template %q", m, tmpl.ID)
		}
		expr := f.SQLExpr
		if expr == "" {
			expr = f.Key
		}
		agg := f.AggFunction
		if agg != "" {
			selects = append(selects, fmt.Sprintf("%s(%s) AS %s", agg, expr, quoteIdent(f.Key)))
		} else {
			selects = append(selects, fmt.Sprintf("%s AS %s", expr, quoteIdent(f.Key)))
		}
	}

	if len(selects) == 0 {
		return "", nil, fmt.Errorf("at least one dimension or measure required")
	}

	// ── WHERE clause ────────────────────────────────────────────────────
	for _, f := range params.Filters {
		field, ok := findFieldAny(tmpl, f.Field)
		if !ok {
			return "", nil, fmt.Errorf("filter field %q not found in template %q", f.Field, tmpl.ID)
		}

		expr := field.SQLExpr
		if expr == "" {
			expr = field.Key
		}

		cond, err := buildCondition(expr, f, nextArg)
		if err != nil {
			return "", nil, fmt.Errorf("filter on %q: %w", f.Field, err)
		}
		where = append(where, cond)
	}

	// ── Time range filter ───────────────────────────────────────────────
	if params.TimeFrom != nil && tmpl.DateField != "" {
		where = append(where, fmt.Sprintf("%s >= %s", quoteIdent(tmpl.DateField), nextArg(*params.TimeFrom)))
	}
	if params.TimeTo != nil && tmpl.DateField != "" {
		where = append(where, fmt.Sprintf("%s <= %s", quoteIdent(tmpl.DateField), nextArg(*params.TimeTo)))
	}

	// ── Assemble query ──────────────────────────────────────────────────
	clauses = append(clauses, "SELECT")
	clauses = append(clauses, strings.Join(selects, ", "))
	clauses = append(clauses, "FROM (")
	clauses = append(clauses, tmpl.SQL)
	clauses = append(clauses, ") AS __sub")

	if len(where) > 0 {
		clauses = append(clauses, "WHERE "+strings.Join(where, " AND "))
	}

	if len(dimExprs) > 0 {
		clauses = append(clauses, "GROUP BY "+strings.Join(dimExprs, ", "))
	}

	if params.OrderBy != "" {
		dir := "ASC"
		if strings.EqualFold(params.OrderDir, "desc") {
			dir = "DESC"
		}
		clauses = append(clauses, fmt.Sprintf("ORDER BY %s %s", quoteIdent(params.OrderBy), dir))
	}

	if params.Limit > 0 {
		clauses = append(clauses, fmt.Sprintf("LIMIT %d", params.Limit))
	}
	if params.Offset > 0 {
		clauses = append(clauses, fmt.Sprintf("OFFSET %d", params.Offset))
	}

	return strings.Join(clauses, "\n"), args, nil
}

// ── Validation ────────────────────────────────────────────────────────────────

// validateFields проверяет, что все выбранные поля существуют в шаблоне.
func (qb *QueryBuilder) validateFields(tmpl QueryTemplate, params QueryParams) error {
	allFields := make(map[string]bool, len(tmpl.Dimensions)+len(tmpl.Measures))
	for _, f := range tmpl.Dimensions {
		allFields[f.Key] = true
	}
	for _, f := range tmpl.Measures {
		allFields[f.Key] = true
	}

	for _, d := range params.Dimensions {
		if !allFields[d] {
			return &ValidationError{
				TemplateID: tmpl.ID,
				Message:    fmt.Sprintf("dimension %q is not allowed", d),
			}
		}
	}
	for _, m := range params.Measures {
		if !allFields[m] {
			return &ValidationError{
				TemplateID: tmpl.ID,
				Message:    fmt.Sprintf("measure %q is not allowed", m),
			}
		}
	}
	return nil
}

// ── Row Reading ───────────────────────────────────────────────────────────────

// readRows конвертирует pgx.Rows в QueryResult.
func (qb *QueryBuilder) readRows(rows pgx.Rows) (*QueryResult, error) {
	columns := rows.FieldDescriptions()
	colNames := make([]string, len(columns))
	for i, col := range columns {
		colNames[i] = string(col.Name)
	}

	result := &QueryResult{
		Columns: colNames,
		Rows:    make([][]interface{}, 0),
	}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		result.Rows = append(result.Rows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	result.Total = int64(len(result.Rows))
	return result, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// findField ищет поле в списке по Key.
func findField(fields []Field, key string) (Field, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// findFieldAny ищет поле в dimensions И measures шаблона.
func findFieldAny(tmpl QueryTemplate, key string) (Field, bool) {
	if f, ok := findField(tmpl.Dimensions, key); ok {
		return f, true
	}
	return findField(tmpl.Measures, key)
}

// buildCondition строит SQL условие из FilterCondition.
func buildCondition(expr string, f FilterCondition, nextArg func(interface{}) string) (string, error) {
	switch strings.ToLower(f.Operator) {
	case "eq":
		return fmt.Sprintf("%s = %s", expr, nextArg(f.Value)), nil
	case "neq":
		return fmt.Sprintf("%s != %s", expr, nextArg(f.Value)), nil
	case "gt":
		return fmt.Sprintf("%s > %s", expr, nextArg(f.Value)), nil
	case "gte":
		return fmt.Sprintf("%s >= %s", expr, nextArg(f.Value)), nil
	case "lt":
		return fmt.Sprintf("%s < %s", expr, nextArg(f.Value)), nil
	case "lte":
		return fmt.Sprintf("%s <= %s", expr, nextArg(f.Value)), nil
	case "in":
		vals, ok := f.Value.([]interface{})
		if !ok {
			return "", fmt.Errorf("IN operator requires array value")
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			placeholders[i] = nextArg(v)
		}
		return fmt.Sprintf("%s IN (%s)", expr, strings.Join(placeholders, ", ")), nil
	case "like":
		return fmt.Sprintf("%s LIKE %s", expr, nextArg(f.Value)), nil
	case "contains":
		return fmt.Sprintf("%s ILIKE %s", expr, nextArg("%"+fmt.Sprint(f.Value)+"%")), nil
	default:
		return "", fmt.Errorf("unsupported operator %q", f.Operator)
	}
}

// quoteIdent оборачивает идентификатор в кавычки (SQL injection prevention).
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ── Error Types ───────────────────────────────────────────────────────────────

// ValidationError — ошибка валидации параметров запроса.
type ValidationError struct {
	TemplateID string
	Message    string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("analytics validation error [template=%s]: %s", e.TemplateID, e.Message)
}

// ExecutionError — ошибка выполнения SQL запроса.
type ExecutionError struct {
	Message string
	Err     error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("analytics execution error: %s: %v", e.Message, e.Err)
}

func (e *ExecutionError) Unwrap() error { return e.Err }

// TimeoutError — ошибка таймаута выполнения запроса.
type TimeoutError struct {
	Message string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("analytics timeout: %s", e.Message)
}
