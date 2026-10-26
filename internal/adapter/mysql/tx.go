package mysql

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"secunda/internal/usecase"
)

type txKey struct{}

var _ usecase.TransactionManager = (*TransactionManager)(nil)

// TransactionManager кладёт активную транзакцию в контекст.
// Все репозитории через ext(ctx, db) автоматически используют её.
type TransactionManager struct {
	db *sqlx.DB
}

func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		// Вложенный вызов присоединяется к уже открытой транзакции.
		return fn(ctx)
	}

	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	return nil
}

func ext(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}
