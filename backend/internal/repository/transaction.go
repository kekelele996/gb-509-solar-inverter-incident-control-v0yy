package repository

import (
	"context"

	"gorm.io/gorm"
)

type contextKey string

const txContextKey contextKey = "gorm-transaction"

// TransactionManager runs a unit of work in one database transaction. Repositories
// participating in the context use the transaction instead of their root handle.
type TransactionManager interface {
	InTx(context.Context, func(context.Context) error) error
}

type transactionManager struct{ db *gorm.DB }

func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &transactionManager{db: db}
}

func (m *transactionManager) InTx(ctx context.Context, fn func(context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txContextKey, tx))
	})
}

func dbFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}
