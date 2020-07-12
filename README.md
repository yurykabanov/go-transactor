# Transactor

Transactor is a simple transaction manager that handles opening/committing transactions.

## Usage

```go
package main

import (
    "context"
    
    "github.com/jmoiron/sqlx"
    "github.com/yurykabanov/go-transactor"
)
   
var (
  db *sqlx.DB
)

func doSomething(ctx context.Context) error {
  var data struct{}

  // Transaction passed implicitly can be extracted via transactor.ExtractTx.
  tx := transactor.ExtractTx(ctx)

  // It is useful to write methods that could be called either in transaction
  // or without explicit one while keeping function signature as simple as possible.
  if tx == nil {
    return tx.GetContext(ctx, &data, "SELECT * FROM some_table LIMIT 1")
  } else {
    return db.GetContext(ctx, &data, "SELECT * FROM some_table LIMIT 1")
  }
}       

func main() {
  db, _ = sqlx.Open("driver_name", "dsn")

  txm := transactor.NewTxManager(db)

  err = txm.WithinTx(context.Background(), func(ctx context.Context, tx *sqlx.Tx) error {
    // 1. Explicitly passed transaction
    var data struct{}
    err := tx.GetContext(ctx, &data, "SELECT * FROM some_table LIMIT 1")
    if err != nil {
      return err
    }

    // 2. Implicitly passed transaction (can be obtained via transactor.ExtractTx)  
    return doSomething(ctx)
  })
  if err != nil {
    panic(err)
  }
}
```
