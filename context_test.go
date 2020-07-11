package transactor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTx_Success(t *testing.T) {
	mock, dbx, _ := createTxManager(t)
	defer dbx.Close()

	mock.ExpectBegin()

	tx, err := dbx.Beginx()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a transaction", err)
	}

	ctx := context.WithValue(context.Background(), implicitTransactionContextKey, tx)
	assert.Equal(t, tx, ExtractTx(ctx))
}

func TestExtractTx_Failure(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, ExtractTx(ctx))
}
