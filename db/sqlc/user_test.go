package db

import (
	"context"
	"simplebank/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 为了保证每个测试单元的独立性，删改查时都应该自行单独创建数据
func CreateRandomUser(t *testing.T) User {
	arg := CreateUserParams{
		Username:       utils.RandomOwner(),
		HashedPassword: "123456",
		FullName:       utils.RandomOwner(),
		Email:          utils.RandomEmal(),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)

	// 是否无报错
	require.NoError(t, err)
	// 返回结果不为空
	require.NotEmpty(t, user)

	// 返回结果值正确
	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	// 返回结果非零值
	require.NotZero(t, user.CreatedAt)
	require.Zero(t, user.PasswordChangedAt)

	return user
}

func TestCreateuser(t *testing.T) {
	CreateRandomUser(t)
}

func TestGetuser(t *testing.T) {
	user1 := CreateRandomUser(t)

	user2, err := testQueries.GetUser(context.Background(), user1.Username)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)

	// 两个创建时间差距应在1秒内
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
	require.Equal(t, user1.PasswordChangedAt, user2.PasswordChangedAt)
}
