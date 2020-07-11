package transactor

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type contextKey uint

const (
	implicitTransactionContextKey contextKey = iota
)

func withTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, implicitTransactionContextKey, tx)
}

func ExtractTx(ctx context.Context) *sqlx.Tx {
	if v := ctx.Value(implicitTransactionContextKey); v != nil {
		if tx, ok := v.(*sqlx.Tx); ok {
			return tx
		}
	}

	return nil
}
