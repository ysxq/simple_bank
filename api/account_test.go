package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	mockdb "simplebank/db/mock"
	db "simplebank/db/sqlc"
	"simplebank/utils"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestGetAccountAPI(t *testing.T) {
	account := randomAccount()

	// 全部测试案例
	testCases := []struct {
		name          string                        // 每个测试案例都有单独的名称
		accountID     int64                         // 测试需要的 account_id
		buildStubs    func(store *mockdb.MockStore) // 声明本次模拟请求 CRUD 的预期结果
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)). // 检查 GetAccount 方法的传参
					Times(1).                                        // GetAccount 方法只会被调用1次
					Return(account, nil)                             // 指定 CRUD 返回内容
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccount(t, recorder.Body, account)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows) // 返回一个空信息和 sql 结果为空的报错
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "BadRequest",
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone) // 返回一个空信息和数据库连接失败
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		// 每个测试案例使用子测试运行
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			// 声明本次模拟请求预期结果
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})
	}
}

func randomAccount() db.Account {
	return db.Account{
		ID:       utils.RandomInt(1, 1000),
		Owner:    utils.RandomOwner(),
		Balance:  utils.RandomMoney(),
		Currency: utils.RandomCurrency(),
	}
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, account db.Account) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotAccount db.Account
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account, gotAccount)
}

func TestCreateAccountAPI(t *testing.T) {
	account := randomAccount()

	// 全部测试案例
	testCases := []struct {
		name          string                        // 每个测试案例都有单独的名称
		requestData   db.CreateAccountParams        // 测试案例请求体内容
		buildStubs    func(store *mockdb.MockStore) // 声明本次模拟请求 CRUD 的预期结果
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			requestData: db.CreateAccountParams{
				Owner:    account.Owner,
				Currency: account.Currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.CreateAccountParams{})).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "BarRequest",
			requestData: db.CreateAccountParams{
				Owner: account.Owner,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.CreateAccountParams{})).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			requestData: db.CreateAccountParams{
				Owner:    account.Owner,
				Currency: account.Currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.CreateAccountParams{})).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		// 每个测试案例使用子测试运行
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			// 声明本次模拟请求预期结果
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := "/accounts"
			body, err := json.Marshal(&tc.requestData)
			require.NoError(t, err)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountAPI(t *testing.T) {
	n := 10
	listAccount := make([]db.Account, 0, n)
	for i := 0; i < n; i++ {
		listAccount = append(listAccount, randomAccount())
	}

	testCases := []struct {
		name          string
		pageId        int32
		pageSize      int32
		buildStubs    func(store *mockdb.MockStore, pageId int32, pageSize int32)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder, pageId int32, pageSize int32)
	}{
		{
			name:     "OK",
			pageId:   1,
			pageSize: 5,
			buildStubs: func(store *mockdb.MockStore, pageId int32, pageSize int32) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.AssignableToTypeOf(db.ListAccountsParams{})).
					Times(1).
					Return(listAccount[(pageId-1)*pageSize:pageSize], nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, pageId int32, pageSize int32) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Equal(t, unmarshalListAccount(t, recorder), listAccount[(pageId-1)*pageSize:pageSize])
			},
		},
		{
			name:     "BadRequest",
			pageId:   0,
			pageSize: 5,
			buildStubs: func(store *mockdb.MockStore, pageId int32, pageSize int32) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, pageId int32, pageSize int32) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:     "InternalError",
			pageId:   1,
			pageSize: 5,
			buildStubs: func(store *mockdb.MockStore, pageId int32, pageSize int32) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.AssignableToTypeOf(db.ListAccountsParams{})).
					Times(1).
					Return([]db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, pageId int32, pageSize int32) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		// 每个测试案例使用子测试运行
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			// 声明本次模拟请求预期结果
			tc.buildStubs(store, tc.pageId, tc.pageSize)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts?page_id=%d&page_size=%d", tc.pageId, tc.pageSize)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder, tc.pageId, tc.pageSize)
		})
	}
}

func unmarshalListAccount(t *testing.T, recorder *httptest.ResponseRecorder) []db.Account {
	body, err := ioutil.ReadAll(recorder.Body)
	require.NoError(t, err)

	var listAccount []db.Account
	err = json.Unmarshal(body, &listAccount)
	require.NoError(t, err)
	return listAccount
}

func unmarshalAccount(t *testing.T, recorder *httptest.ResponseRecorder) db.Account {
	body, err := ioutil.ReadAll(recorder.Body)
	require.NoError(t, err)

	var account db.Account
	err = json.Unmarshal(body, &account)
	require.NoError(t, err)
	return account
}

func TestDeleteAccountAPI(t *testing.T) {
	account := randomAccount()

	testCases := []struct {
		name          string
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "BadRequest",
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		// 每个测试案例使用子测试运行
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			// 声明本次模拟请求预期结果
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}
}

func TestUpdateAccountAPI(t *testing.T) {
	account1 := randomAccount()
	account2 := account1
	account2.Balance += 10

	testCases := []struct {
		name          string
		requestData   updateAccountRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			requestData: updateAccountRequest{
				Id:      account1.ID,
				Balance: account2.Balance,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.UpdateAccountParams{})).
					Times(1).
					Return(account2, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Equal(t, account2, unmarshalAccount(t, recorder))
			},
		},
		{
			name: "NotFound",
			requestData: updateAccountRequest{
				Id:      account1.ID,
				Balance: account2.Balance,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.UpdateAccountParams{})).
					Times(1).
					Return(account2, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "BadRequest",
			requestData: updateAccountRequest{
				Id:      0,
				Balance: account2.Balance,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.UpdateAccountParams{})).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			requestData: updateAccountRequest{
				Id:      account1.ID,
				Balance: account2.Balance,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.AssignableToTypeOf(db.UpdateAccountParams{})).
					Times(1).
					Return(account2, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		// 每个测试案例使用子测试运行
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			// 声明本次模拟请求预期结果
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := "/accounts"
			body, err := json.Marshal(&tc.requestData)
			require.NoError(t, err)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}
}