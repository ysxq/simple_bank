package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDb)

	account1 := CreateRandomAccount(t)
	account2 := CreateRandomAccount(t)
	fmt.Println(">> befor:", account1.Balance, account2.Balance)

	// 使用并发验证事务操作
	n := 5
	amount := int64(10)

	// 使用 chan 接收 goroutine 结果，确保 testify/require 能够正常停止本次测试
	errs := make(chan error)
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		go func() {
			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountId: account1.ID,
				ToAccountId:   account2.ID,
				Amount:        amount,
			})
			errs <- err
			results <- result
		}()
	}

	existed := make(map[int]bool)
	// 验证结果
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// check accounts
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)

		fmt.Println(">> tx:", fromAccount.Balance, toAccount.Balance)

		// check accounts' balance
		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)

		// 每次转账完成后的金额差值，应该是转账金额的倍数（因为每次转账金额相同）
		require.True(t, diff1%amount == 0) // 1 * amount, 2 * amount, 3 * amount, ..., n * amount

		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check final updated balances
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)

	require.Equal(t, account1.Balance-amount*int64(n), updatedAccount1.Balance)
	require.Equal(t, account2.Balance+amount*int64(n), updatedAccount2.Balance)
}

// 模拟死锁情况
func TestTransferTxDeadLock(t *testing.T) {
	store := NewStore(testDb)

	account1 := CreateRandomAccount(t)
	account2 := CreateRandomAccount(t)

	// 使用并发验证事务操作
	n := 10
	amount := int64(10)

	// 使用 chan 接收 goroutine 结果，确保 testify/require 能够正常停止本次测试
	errs := make(chan error)

	for i := 0; i < n; i++ {
		// 一半的交易，1 => 2 转钱，另一半的交易，2 => 1 转钱
		fromAccountID := account1.ID
		toAccountId := account2.ID

		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountId = account1.ID
		}

		go func() {
			_, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountId: fromAccountID,
				ToAccountId:   toAccountId,
				Amount:        amount,
			})
			errs <- err
		}()
	}

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	// check final updated balances
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	// 因为1和2互转，所以两者余额应该不变
	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}
