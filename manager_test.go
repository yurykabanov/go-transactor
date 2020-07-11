package transactor

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func createTxManager(t *testing.T) (sqlmock.Sqlmock, *sqlx.DB, *TxManager) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dbx := sqlx.NewDb(db, "mysql")

	txm := NewTxManager(dbx)

	return mock, dbx, txm
}

func TestTxManager_WithinTx_Success_ContextValuesArePassed(t *testing.T) {
	mock, dbx, txm := createTxManager(t)
	defer dbx.Close()

	baseContext := context.WithValue(context.Background(), "some_key", "some_value")

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := txm.WithinTx(baseContext, func(ctx context.Context, tx *sqlx.Tx) error {
		assert.Equal(t, "some_value", ctx.Value("some_key"), "context should be passed through")
		assert.Equal(t, tx, ctx.Value(implicitTransactionContextKey), "transaction should be set in passed context")

		return nil
	})
	assert.Nil(t, err, "successful commit")

	mockErr := mock.ExpectationsWereMet()
	assert.Nil(t, mockErr, "there should not be any unfulfilled expectations")
}

func TestTxManager_WithinTx_NestedCallsShouldNotOpenNewTx(t *testing.T) {
	mock, dbx, txm := createTxManager(t)
	defer dbx.Close()

	baseContext := context.Background()

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := txm.WithinTx(baseContext, func(ctx context.Context, outerTx *sqlx.Tx) error {
		return txm.WithinTx(ctx, func(ctx context.Context, innerTx *sqlx.Tx) error {
			assert.Equal(t, outerTx, innerTx)
			return nil
		})
	})
	assert.Nil(t, err, "successful commit")

	mockErr := mock.ExpectationsWereMet()
	assert.Nil(t, mockErr, "there should not be any unfulfilled expectations")
}

func TestTxManager_WithinTx_PanicShouldCauseRollbackAndPanic(t *testing.T) {
	mock, dbx, txm := createTxManager(t)
	defer dbx.Close()

	baseContext := context.Background()

	mock.ExpectBegin()
	mock.ExpectRollback()

	assert.Panicsf(t, func() {
		_ = txm.WithinTx(baseContext, func(ctx context.Context, tx *sqlx.Tx) error {
			panic("panic in transaction!")
		})
	}, "panic in transaction!", "panic should be recovered, but called again after recover")

	mockErr := mock.ExpectationsWereMet()
	assert.Nil(t, mockErr, "there should not be any unfulfilled expectations")
}

func TestTxManager_WithinTx_ErrorShouldCauseRollback(t *testing.T) {
	mock, dbx, txm := createTxManager(t)
	defer dbx.Close()

	baseContext := context.Background()

	mock.ExpectBegin()
	mock.ExpectRollback()

	someError := errors.New("some error occurred")

	err := txm.WithinTx(baseContext, func(ctx context.Context, tx *sqlx.Tx) error {
		return someError
	})
	assert.Equal(t, someError, err, "error should be returned as is")

	mockErr := mock.ExpectationsWereMet()
	assert.Nil(t, mockErr, "there should not be any unfulfilled expectations")
}

func TestTxManager_WithinTx_BeginTxError(t *testing.T) {
	mock, dbx, txm := createTxManager(t)
	defer dbx.Close()

	baseContext := context.Background()

	beginTxErr := errors.New("error on begin tx")
	mock.ExpectBegin().WillReturnError(beginTxErr)

	err := txm.WithinTx(baseContext, func(ctx context.Context, tx *sqlx.Tx) error {
		return nil
	})
	assert.Equal(t, beginTxErr, err, "begin tx error should be returned")

	mockErr := mock.ExpectationsWereMet()
	assert.Nil(t, mockErr, "there should not be any unfulfilled expectations")
}

func TestTxManager_WithinTx_CommitTxError(t *testing.T) {
	mock, dbx, txm := createTxManager(t)
	defer dbx.Close()

	baseContext := context.Background()

	commitTxErr := errors.New("error on begin tx")
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(commitTxErr)

	err := txm.WithinTx(baseContext, func(ctx context.Context, tx *sqlx.Tx) error {
		return nil
	})
	assert.Equal(t, commitTxErr, err, "commit tx error should be returned")

	mockErr := mock.ExpectationsWereMet()
	assert.Nil(t, mockErr, "there should not be any unfulfilled expectations")
}
