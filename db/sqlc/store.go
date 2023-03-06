package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store 提供了所有转账相关方法
type Store struct {
	*Queries         // 组合 sqlc 生成的单个数据库操作
	db       *sql.DB // 用于执行事务
}

// NewStore creates a new Store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx 使用事务执行一个数据库操作的方法
func (stroe *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	// 开始事务，使用默认的读隔离
	tx, err := stroe.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// 执行数据库操作
	q := New(tx)
	err = fn(q)
	if err != nil {
		// 数据库转账操作失败，事务回退
		// 如果事务回退也报错，将两个报错合并返回
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}

		return err
	}

	// 事务操作成功，提交事务
	return tx.Commit()
}

// 转账所需参数
type TransferTxParams struct {
	FromAccountId int64 `json:"from_account_id"`
	ToAccountId   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// 转账操作所有创建的数据库数据
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// 使用事务执行转账操作
// 包含创建转账记录、扣账记录、入账记录、账户扣账、账户入账
func (Store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := Store.execTx(ctx, func(q *Queries) error {
		var err error

		// 创建转账记录
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountId,
			ToAccountID:   arg.ToAccountId,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		// 扣账记录
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountId,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		// 入账记录
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountId,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// 规避死锁问题，让 id 值更小的账户先执行
		if arg.FromAccountId < arg.ToAccountId {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountId, -arg.Amount, arg.ToAccountId, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountId, arg.Amount, arg.FromAccountId, -arg.Amount)
		}

		return err
	})

	return result, err
}

// 实现两个账户的余额操作（转账）
func addMoney(ctx context.Context, q *Queries, accountID1, amount1, accountID2, amount2 int64) (account1, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})

	return
}
