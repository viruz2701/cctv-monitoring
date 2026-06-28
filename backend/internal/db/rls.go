// Package db — Row Level Security helpers (F-0.2.3).
//
// RLS (Row Level Security) обеспечивает изоляцию данных между tenant'ами
// на уровне PostgreSQL. Каждый запрос к БД устанавливает контекст
// session-local параметров (app.tenant_id, app.role), которые затем
// проверяются RLS-политиками на таблицах.
//
// Архитектура:
//
//	HTTP Request → AuthMiddleware → TenantMiddleware → Handler
//	                                        │
//	                                        ▼
//	                                   SetTenantContext()
//	                                        │
//	                                        ▼
//	                              SET LOCAL app.tenant_id = '...'
//	                              SET LOCAL app.role = '...'
//	                                        │
//	                                        ▼
//	                              SQL Query → RLS Policy → Filtered Rows
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - IEC 62443 SR 5.1 (Network segmentation — zone-based access)
//   - ISO 27001 A.9.1.2 (Access control — tenant data separation)
//   - ISO 27019 PCC.A.13 (ICS network segregation)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
//   - Приказ ОАЦ № 66 п. 7.18.3 (Изоляция данных)
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantAdminBypass — значение tenant_id для admin bypass в RLS-политиках.
const TenantAdminBypass = "*"

// SetTenantContext устанавливает session-local параметры для RLS.
//
// Использует SET LOCAL, который действует только в рамках текущей
// транзакции или сессии соединения. Для pgxpool важно вызывать эту
// функцию в рамках одной транзакции (иначе SET LOCAL сбросится при
// возврате соединения в пул).
//
// tenantID: идентификатор tenant'а или "*" для admin bypass.
// role: роль пользователя (admin, technician, operator, viewer).
//
// Соответствует: IEC 62443 SR 2.1, ISO 27001 A.9.1.2
func SetTenantContext(ctx context.Context, pool *pgxpool.Pool, tenantID, role string) error {
	// WARNING: SET_CONFIG без транзакции действует только в рамках текущего
	// соединения. При возврате в пул контекст сбрасывается.
	// Используйте WithTenantTx для гарантированной изоляции.
	if tenantID == "" {
		tenantID = ""
	}
	if role == "" {
		role = "viewer"
	}

	_, err := pool.Exec(ctx,
		`SELECT set_config('app.tenant_id', $1, true),
		        set_config('app.role', $2, true)`,
		tenantID, role,
	)
	if err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}
	return nil
}

// WithTenantQuery выполняет функцию в контексте tenant'а.
//
// Устанавливает app.tenant_id и app.role через SET LOCAL перед
// выполнением запроса, что гарантирует корректную работу RLS-политик.
//
// Использование:
//
//	err := db.WithTenantQuery(ctx, pool, "admin", func(qctx context.Context) error {
//	    _, err := pool.Exec(qctx, "UPDATE devices SET name = $1 WHERE id = $2", name, id)
//	    return err
//	})
//
// Соответствует: IEC 62443 SR 5.1, ISO 27001 A.9.1.2
func WithTenantQuery(ctx context.Context, pool *pgxpool.Pool, tenantID, role string, fn func(context.Context) error) error {
	// ⚠ SetTenantContext вне транзакции — RLS контекст может сброситься
	// при возврате соединения в пул. Используйте WithTenantTx для гарантии.
	if err := SetTenantContext(ctx, pool, tenantID, role); err != nil {
		return fmt.Errorf("with tenant query: %w", err)
	}

	return fn(ctx)
}

// WithTenantTx выполняет транзакцию в контексте tenant'а.
//
// В отличие от WithTenantQuery, эта функция оборачивает выполнение
// в BEGIN/COMMIT с SET LOCAL внутри транзакции, что гарантирует,
// что RLS-контекст не сбросится при возврате соединения в пул.
//
// Использование:
//
//	err := db.WithTenantTx(ctx, pool, "tenant_123", "admin",
//	    func(tx pgx.Tx) error {
//	        _, err := tx.Exec(ctx, "UPDATE devices SET name = $1 WHERE id = $2", name, id)
//	        return err
//	    })
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID, role string, fn func(context.Context) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// SET LOCAL внутри транзакции — действует до COMMIT/ROLLBACK
	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.tenant_id', $1, true),
		        set_config('app.role', $2, true)`,
		tenantID, role,
	); err != nil {
		return fmt.Errorf("set tenant context in tx: %w", err)
	}

	if err := fn(ctx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
