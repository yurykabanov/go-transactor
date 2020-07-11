package transactor

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
)

// TxManager is a simple transaction manager that handles opening/committing
// transactions.
type TxManager struct {
	db *sqlx.DB
}

func NewTxManager(db *sqlx.DB) *TxManager {
	return &TxManager{
		db: db,
	}
}

// TxFunc is an alias for function passed to TxManager.WithinTx.
type TxFunc func(ctx context.Context, tx *sqlx.Tx) error

// WithinTx executes txFunc within transaction opening new transaction and
// committing it.
//
// Passes transaction implicitly via context. Implicit transaction can be
// obtained using ExtractTx function.
//
// txFunc should not commit or rollback passed transaction. txFunc can either use
// the transaction that is explicitly provided or use the transaction that is
// implicitly provided via context (through ExtractTx function), i.e.
//   txm.WithinTx(originalCtx, func(ctx context.Context, tx *sqlx.Tx) error {
//     tx.DoSomething() // ok
//     ExtractTx(ctx).DoSomething() // ok
//   })
//
// Nested calls neither open new transactions, nor close existing one.
// Transaction will be committed (or rollbacked) when txFunc of the outermost
// WithinTx call is finished, i.e.
//   txm.WithinTx(originalCtx, func(outerCtx context.Context, outerTx *sqlx.Tx) error {
//     // before: new tx is opened and stored in ctx
//
//     // outerCtx := originalCtx + tx
//
//     txm.WithinTx(outerCtx, func(innerCtx context.Context, innerTx *sqlx.Tx) error {
//       // before: no new tx is opened
//
//       // innerCtx == outerCtx - due to nested call
//       // innerTx == outerTx - due to nested call
//
//       // after: tx is not committed (or rollbacked)
//     })
//
//     // after: tx is committed (or rollbacked)
//   })
func (txm *TxManager) WithinTx(ctx context.Context, txFunc TxFunc) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "transactor.TxManager::WithinTx")
	defer span.Finish()
	defer tagSpanWithError(span, &err)

	var tx *sqlx.Tx

	tx = ExtractTx(ctx)

	if tx == nil {
		span.LogKV("message", "tx: creating new")

		tx, err = txm.db.Beginx()
		if err != nil {
			return
		}

		ctx = withTx(ctx, tx)

		defer func() {
			if p := recover(); p != nil {
				span.LogKV("message", "tx: rollback due to panic")

				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					span.SetTag("rollback_error", true).LogKV("rollback_error_message", rollbackErr)
				}

				panic(p)
			} else if err != nil {
				span.LogKV("message", "tx: rollback due to error")

				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					span.SetTag("rollback_error", true).LogKV("rollback_error_message", rollbackErr)
				}

			} else {
				span.LogKV("message", "tx: commit")

				err = tx.Commit()
			}
		}()
	} else {
		span.LogKV("message", "tx: using existing")
	}

	err = txFunc(ctx, tx)

	return err
}
