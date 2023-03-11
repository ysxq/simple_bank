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
	"simplebank/token"
	"simplebank/utils"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCreateTransfer(t *testing.T) {
	user, _ := randomUser(t)
	transferTxResult := randomTransferTxResult(t, user.Username)

	// 全部测试案例
	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"from_account_id": transferTxResult.FromAccount.ID,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.FromAccount.ID)).
					Times(1).
					Return(transferTxResult.FromAccount, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.ToAccount.ID)).
					Times(1).
					Return(transferTxResult.ToAccount, nil)

				arg := db.TransferTxParams{
					FromAccountId: transferTxResult.Transfer.FromAccountID,
					ToAccountId:   transferTxResult.Transfer.ToAccountID,
					Amount:        transferTxResult.Transfer.Amount,
				}

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(transferTxResult, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTransferTxResult(t, recorder.Body, transferTxResult)
			},
		},
		{
			name: "UnauthorizedUser",
			body: gin.H{
				"from_account_id": transferTxResult.FromAccount.ID,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "unauthorized_user", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.FromAccount.ID)).
					Times(1).
					Return(transferTxResult.FromAccount, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0).
					Return(transferTxResult, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "BadRequest1",
			body: gin.H{
				"from_account_id": 0,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequest2",
			body: gin.H{
				"from_account_id": transferTxResult.FromAccount.ID,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.USD,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.FromAccount.ID)).
					Times(1).
					Return(transferTxResult.FromAccount, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "NotFound",
			body: gin.H{
				"from_account_id": 100000,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError1",
			body: gin.H{
				"from_account_id": transferTxResult.FromAccount.ID,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.FromAccount.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InternalError2",
			body: gin.H{
				"from_account_id": transferTxResult.FromAccount.ID,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.FromAccount.ID)).
					Times(1).
					Return(transferTxResult.FromAccount, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.ToAccount.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InternalError3",
			body: gin.H{
				"from_account_id": transferTxResult.FromAccount.ID,
				"to_account_id":   transferTxResult.ToAccount.ID,
				"amount":          transferTxResult.Transfer.Amount,
				"currency":        utils.RMB,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.FromAccount.ID)).
					Times(1).
					Return(transferTxResult.FromAccount, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(transferTxResult.ToAccount.ID)).
					Times(1).
					Return(transferTxResult.ToAccount, nil)

				arg := db.TransferTxParams{
					FromAccountId: transferTxResult.Transfer.FromAccountID,
					ToAccountId:   transferTxResult.Transfer.ToAccountID,
					Amount:        transferTxResult.Transfer.Amount,
				}

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(db.TransferTxResult{}, sql.ErrConnDone)
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
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := "/transfer"
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}
}

func randomTransferTxResult(t *testing.T, owner string) db.TransferTxResult {
	fromAccount := randomAccount(owner)
	fromAccount.Currency = utils.RMB

	user2, _ := randomUser(t)
	toAccount := randomAccount(user2.Username)
	toAccount.Currency = utils.RMB
	fmt.Println(fromAccount, toAccount)
	amount := int64(10)
	transfer := db.Transfer{
		ID:            utils.RandomInt(1, 100),
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        amount,
	}
	fromEntry := db.Entry{
		ID:        utils.RandomInt(1, 1000),
		AccountID: fromAccount.ID,
		Amount:    -amount,
	}
	toEntry := db.Entry{
		ID:        utils.RandomInt(1, 1000),
		AccountID: toAccount.ID,
		Amount:    amount,
	}

	return db.TransferTxResult{
		Transfer:    transfer,
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		FromEntry:   fromEntry,
		ToEntry:     toEntry,
	}
}

func requireBodyMatchTransferTxResult(t *testing.T, body *bytes.Buffer, transferTxResult db.TransferTxResult) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gottransferTxResult db.TransferTxResult
	err = json.Unmarshal(data, &gottransferTxResult)
	require.NoError(t, err)
	require.Equal(t, transferTxResult, gottransferTxResult)
}
