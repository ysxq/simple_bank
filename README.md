# Simple bank

简易银行系统

视频教程：[Golang + Postgres +Docker\] (中文字幕)_哔哩哔哩_bilibili](https://www.bilibili.com/video/BV1dy4y1u7Sq/?spm_id_from=333.788.recommend_more_video.2&vd_source=8ecdd6dc2d8760b0e21800e13d9ef1f0)

项目地址：[techschool/simplebank: Backend master class: build a simple bank service in Go (github.com)](https://github.com/techschool/simplebank)

架构：Postgres + Docker





# 环境构建

## 一、数据设计

使用 https://dbdiagram.io/ 网页进行数据库设计，并生成 sql 文件

结果：https://dbdiagram.io/d/6401b211296d97641d8529aa



```sql
// 账户
Table accounts as A {
  id bigserial [pk] // 主键，自增的大整数
  owner varchar [not null]
  balance bigint [not null, note:'余额']
  currency varchar [not null, note:'币种']
  created_at timestamptz [not null, default: `now()`] // 包含时区的时间

  Indexes {
    owner
  }
}

// 记录账户余额所有更改
Table entries {
  id bigserial [pk]
  account_id bigint [ref: > A.id, not null] // 关联账户表id
  amount bigint [not null, note:'变更金额，允许正负']
  created_at timestamptz [not null, default: `now()`]

  Indexes {
    account_id
  }
}

// 转账记录
Table transfers {
  id bigserial [pk]
  from_account_id bigint [ref: > A.id, not null]
  to_account_id bigint [ref: > A.id, not null]
  amount bigint [not null, note:'转账金额，必须为正']
  created_at timestamptz [not null, default: `now()`]

  Indexes {
    from_account_id
    to_account_id
    (from_account_id, to_account_id)
  }
}
```



## 二、Postgres

使用 docker 安装 Postgres 数据库

官方：[postgres - Official Image | Docker Hub](https://hub.docker.com/_/postgres)



```shell
# 下载最新镜像
docker pull postgres:15.2-alpine

# 启动
#  创建用户 root、设置密码 123456
docker run --name postgres15 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=123456 -d postgres:15.2-alpine

# 进入数据库
docker exec -it postgres15 psql -U root
```



## 三、数据迁移

文档：[migrate/cmd/migrate at master · golang-migrate/migrate (github.com)](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)



1. 配置环境

```shell
# 安装命令工具
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 查看版本（本次安装无版本信息）
migrate -version

# 创建项目目录
mkdir simplebank
cd simplebank

# 创建迁移对应目录
mkdir -p db/migration

# 创建迁移文件
#  -ext 指定文件后缀
#  -seq 指定文件前缀
migrate create -ext sql -dir db/migration -seq init_schema
#  000001_init_schema.up.sql   执行sql操作，000001是版本号，第一个执行
#  000001_init_schema.down.sql 恢复sql操作，01 最后一个执行

# 使用第一步生成的 sql 文件填充 up 文件
vim db/migration/000001_init_schema.up.sql
```

```sql
CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "balance" bigint NOT NULL,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "entries" (
  "id" bigserial PRIMARY KEY,
  "account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "accounts" ("owner");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");

CREATE INDEX ON "transfers" ("from_account_id", "to_account_id");

COMMENT ON COLUMN "accounts"."balance" IS '余额';

COMMENT ON COLUMN "accounts"."currency" IS '币种';

COMMENT ON COLUMN "entries"."amount" IS '变更金额，允许正负';

COMMENT ON COLUMN "transfers"."amount" IS '转账金额，必须为正';

ALTER TABLE "entries" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id");
```

```shell
# 填充 down 文件
vim db/migration/000001_init_schema.down.sql
```

```sql
DROP TABLE IF EXISTS "transfers"
DROP TABLE IF EXISTS "entries"
DROP TABLE IF EXISTS "accounts"
```

2. 迁移数据（初始化数据）

```shell
# 为了方便操作，直接创建一个 Makfile
vim Makefile
```

```makefile
postgres:
	docker run --name postgres15 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=123456 -d postgres:15.2-alpine

createdb:
	docker exec -it postgres15 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres15 dropdb simple_bank

migrateup:
	migrate -path db/migration -database "postgresql://root:123456@localhost:5432/simple_bank?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:123456@localhost:5432/simple_bank?sslmode=disable" -verbose down

.PHONY: postgres createdb dropdb migrateup migratedown
```

```shell
# 上述命令最好测试一下确保正确

# 迁移数据
make migrateup

# 测试效果
docker exec -it postgres15 psql -U root -d simple_bank

simple_bank-# \d
               List of relations
 Schema |       Name        |   Type   | Owner 
--------+-------------------+----------+-------
 public | accounts          | table    | root
 public | accounts_id_seq   | sequence | root
 public | entries           | table    | root
 public | entries_id_seq    | sequence | root
 public | schema_migrations | table    | root
 public | transfers         | table    | root
 public | transfers_id_seq  | sequence | root
(7 rows)
```



## 四、CRUD

本次使用 sqlc 库进行数据库 CRUD 操作的生成

仓库：[kyleconroy/sqlc: Generate type-safe code from SQL (github.com)](https://github.com/kyleconroy/sqlc)

文档：[PostgreSQL 入门 — sqlc 1.14.0 文档](https://docs.sqlc.dev/en/v1.14.0/tutorials/getting-started-postgresql.html)



1. 安装客户端工具

```shell
go install github.com/kyleconroy/sqlc/cmd/sqlc@latest

# 不过本次使用该命令报错（centos7.9 的依赖库版本太低），所以手动下载
curl -L https://downloads.sqlc.dev/sqlc_1.14.0_linux_amd64.tar.gz | tar zx
mv sqlc /usr/local/go/bin/

# 验证
sqlc version
```



2. 修改 sqlc 配置文件

具体配置参考文档：[配置 — sqlc 1.14.0 文档](https://docs.sqlc.dev/en/v1.14.0/reference/config.html)

```shell
# 初始化
sqlc init

# 创建目录，保存 sqlc 生成内容
mkdir db/{sqlc,query}

# 手动修改生成的配置文件
vim sqlc.yaml
```

```yaml
version: 1
packages:
  - name: "db"                # 生成代码的包名称
    path: "./db/sqlc"         # 生成代码存放目录
    queries: "./db/query/"    # 增删改查的SQL代码定义目录
    schema: "./db/migration/" # SQL迁移目录或单个SQL文件的路径
    engine: "postgresql"
    emit_json_tags: true      # 将 JSON 标记添加到生成的结构体
```

```shell
# sqlc 生成代码命令也放到 Makfile 中
vim Makefile
```

```makefile
# 生成 CRUD 代码
sqlc:
	sqlc generate

.PHONY: postgres createdb dropdb migrateup migratedown sqlc
```



3. 配置SQL

```shell
# 仅记录 account 表，其他表可以查看源码
vim db/query/account.sql
```

```sql
-- name: CreateAccount :one
INSERT INTO accounts (
  owner,
  balance,
  currency
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateAccount :one
UPDATE accounts
SET balance = $2
WHERE id = $1
RETURNING *;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = $1;
```



4. 生成代码

```shell
# 先初始化 go mod
go mod init simplebank

# sqlc 生成代码
make sqlc
```



## 五、数据测试

对 sqlc 生成的 CRUD 代码做测试



1. 测试数据库连接

db/sqlc/main_test.go

```go
package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://root:123456@localhost:5432/simple_bank?sslmode=disable"
)

var testQueries *Queries
var testDb *sql.DB

func TestMain(m *testing.M) {
	var err error

	testDb, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDb)

	os.Exit(m.Run())
}
```



2. 测试 account 表操作

先搞一个工具包，随机生成需要的内容

utils/random.go

```go
package utils

import (
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// return a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// return a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// return a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// return a random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

// return a random currency code
func RandomCurrency() string {
	currencies := []string{"USD", "RMB", "EUR"} // 美元、人民币、欧元
	n := len(currencies)
	return currencies[rand.Intn(n)]
}
```

对 account CRUD 测试

db/sqlc/account_test.go

```go
package db

import (
	"context"
	"simplebank/utils"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	arg := CreateAccountParams{
		Owner:    utils.RandomOwner(),
		Balance:  utils.RandomMoney(),
		Currency: utils.RandomCurrency(),
	}

	account, err := testQueries.CreateAccount(context.Background(), arg)

	// 是否无报错
	require.NoError(t, err)
	// 返回结果不为空
	require.NotEmpty(t, account)

	// 返回结果值正确
	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)
	// 返回结果非零值
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)
}
```

测试操作写到 Makefile 中

```makefile
# 显示日志、显示测试覆盖率、所有包执行单元测试
test:
	go test -v -cover ./...

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
```



## 六、事务实现

实现转账操作，需要五个步骤：

1. 创建一条 transfer 数据，记录转账内容
2. 创建一条 entry 数据，from_account 余额扣除
3. 创建一条 entry 数据，to_account 余额增加
4. account 中 from_account 金额扣除
5. account 中 to_account 金额增加

为了保证钱的安全，所以必须保证**原子性**



### 死锁

而且需要考虑多线程操作时数据库死锁问题：

- 如下俩个事务操作

```sql
# 1 => 2 $10
BEGIN;
UPDATE accounts SET balance = balance - 10 WHERE id = 1 RETURNING *;
UPDATE accounts SET balance = balance + 10 WHERE id = 2 RETURNING *;
COMMIT;

# 2 => 1 $10
BEGIN;
UPDATE accounts SET balance = balance - 10 WHERE id = 2 RETURNING *;
UPDATE accounts SET balance = balance + 10 WHERE id = 1 RETURNING *;
COMMIT;
```

- 如果上述的两个事务同时操作：
    - 第一个事务第一条 SQL 会正常执行，第二个事务第一条 SQL 正常执行
    - 第一个事务第二条 SQL 与第二个事务第二条 SQL 互相锁住，形成死锁

- 如果想规避该问题，可以修改 SQL 执行顺序：

```sql
# 1 => 2 $10
BEGIN;
UPDATE accounts SET balance = balance - 10 WHERE id = 1 RETURNING *;
UPDATE accounts SET balance = balance + 10 WHERE id = 2 RETURNING *;
COMMIT;

# 2 => 1 $10
BEGIN;
UPDATE accounts SET balance = balance + 10 WHERE id = 1 RETURNING *;
UPDATE accounts SET balance = balance - 10 WHERE id = 2 RETURNING *;
COMMIT;
```

- 如果上述的两个事务同时操作：
    - 第一个事务第一条 SQL 正常执行，并形成互斥锁，阻塞第二个事务第一条 SQL
    - 第一个事务第二条 SQL 正常执行，事务执行完毕，互斥锁解除
    - 第二个事务正常执行



### 隔离级别

Postgres 事务隔离只能在事务中设置，仅影响单次事务

- Postgres 默认事务级别为读已提交
- 事务隔离级别由低到高分别为 Read uncommitted（读未提交） 、Read committed （读已提交）、Repeatable read （重复读）、Serializable （序列化）
- 事务导致可能出现的问题有脏读、不可重复读、幻读、序列化异常
- Postgres 事务隔离级别和 Mysql 有所区别
    - Read uncommitted 级别
        - Mysql 事务中可以读取到其他事务修改但未提交的修改
        - Postgres 中 Read uncommitted 与 Read committed 级别相同
    - Repeatable read 级别
        - Mysql 事务中修改数据后提交，另一个事务读取数据不会不可重复读和幻读；但修改数据后会在真实数据基础上修改，与本次事务修改不符
        - Postgres 不会幻读，如果修改数据会报错
    - Serializable 级别
        - Mysql 事务中修改数据，其他事务所有操作会被阻塞，等待第一个事务释放锁；如果两个事务都执行了查询操作，第一个事务的修改会被阻塞，如果第二个事务也想修改会导致死锁
        - Progres 事务不会阻塞读写，但是如果两个事务之间具有读/写依赖性（比如 count()），第二个提交的事务会报错，提示再试一次



```shell
# 查看默认事务级别
simple_bank=# show transaction isolation level;
 transaction_isolation 
-----------------------
 read committed
(1 row)

# 开启事务
simple_bank=# begin;
BEGIN

# 设置隔离级别为
simple_bank=*# set transaction isolation level repeatable read;
SET
```



## 七、Actions

GitHub 提供了 Actions 功能，实现 CI

文档：[Building and testing Go - GitHub Docs](https://docs.github.com/en/actions/automating-builds-and-tests/building-and-testing-go)



```shell
# 创建配置文件
mkdir -p .github/workflows
vim ci.yml
```

```yaml
# This workflow will build a golang project
# For more information see: https://docs.github.com/en/actions/automating-builds-and-tests/building-and-testing-go

name: ci-test

# 定义触发工作流的事件
on:
  push:
    branches: [ "main" ] # 推送到 main 分支
  pull_request:
    branches: [ "main" ] # 合并到 main 分支的请求

jobs:

  test:
    name: Test
    runs-on: ubuntu-latest # 该工作流允许的操作系统

    services: # 运行服务
      postgres:
        image: postgres:15 # docker 镜像
        env:
          POSTGRES_USER: root
          POSTGRES_PASSWORD: 123456
          POSTGRES_DB: simple_bank
        options: >- # 健康检查，postgres 是否正常启动
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
    - uses: actions/checkout@v3 # 复用官方提供的操作，仓库代码部署到运行器中

    - name: Set up Go
      uses: actions/setup-go@v3 # 安装 go1.18
      with:
        go-version: 1.18

    - name: Set up migrate
      run: | # 使用管道符声明使用多行命令
        go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
        which migrate

    - name: Run migrations
      run: make migrateup

    - name: Test
      run: make test

```



## 八、配置读取

GitHub：[spf13/viper: Go configuration with fangs (github.com)](https://github.com/spf13/viper)

使用 viper 读取配置

- 支持 JSON、TOML、YAML、ENV、INI
- 支持命令行参数
- 支持 ETCD
- 支持热加载



app.env

```ini
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:123456@localhost:5432/simple_bank?sslmode=disable
SERVER_ADDRESS=0.0.0.0:8080
```



utils/config.go

```go
package utils

import "github.com/spf13/viper"

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	DBDriver      string `mapstructure:"DB_DRIVER"`
	DBSource      string `mapstructure:"DB_SOURCE"`
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
}

// LoadConig reads configuration from config file or environment variables.
func LoadConig(path string) (config Config, err error) {
	viper.AddConfigPath(path)  // 配置文件所在目录
	viper.SetConfigName("app") // 配置文件名称（不包含后缀）
	viper.SetConfigType("env") // 配置文件类型（后缀）

	viper.AutomaticEnv() // 尝试加载环境变量中的配置信息

	err = viper.ReadInConfig() // 读取配置信息
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
```



main.go

```go
conf, err := utils.LoadConig(".")
if err != nil {
    log.Fatal("cannot load config:", err)
}
```







# RESTful API

使用 gin 框架



## API 测试

GitHub：[golang/mock: GoMock is a mocking framework for the Go programming language. (github.com)](https://github.com/golang/mock)

使用 gomock 模拟数据库接口，并对 API 编写单元测试



1. 安装 mockgen 工具

```shell
go install github.com/golang/mock/mockgen@v1.6.0

# 当前项目添加 gomock 依赖
go get github.com/golang/mock
```

2. 对实现 CRUD 的接口自动映射，实现 mock CRUD
    - 目前使用 Store 结构体实现 CRUD，只能连接到真实数据库
    - 可以将 Store 改造为接口



3. mockgen 生成 mock 代码

```shell
mockgen -package mockdb -destination db/mock/store.go simplebank/db/sqlc Store
# -package mockdb    指定生成文件的包名
# -destination ...   指定生成文件位置
# simplebank/db/sqlc 指定CRUD接口所在目录
# Store              指定CRUD接口名称
```





## gin 自定义验证器

utils/currency.go

```go
package utils

// Constants for all supported currencies
const (
	USD = "USD"
	RMB = "RMB"
	EUR = "EUR"
)

// IsSupportCurrency returns true if the currency is supported
func IsSupportCurrency(currency string) bool {
	switch currency {
	case USD, RMB, EUR:
		return true
	}

	return false
}
```



api/validator.go

```go
package api

import (
	"simplebank/utils"

	"github.com/go-playground/validator/v10"
)

var validCurrency validator.Func = func(fl validator.FieldLevel) bool {
	// 先判断数据是否为字符串
	if currency, ok := fl.Field().Interface().(string); ok {
		return utils.IsSupportCurrency(currency)
	}
	return false
}
```



api/server.go

```go
package api

import (
	db "simplebank/db/sqlc"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Server servers HTTP requests for our banking service.
type Server struct {
	store  db.Store
	router *gin.Engine
}

// NewServer creates a new HTTP server and setup routing.
func NewServer(store db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// 注册 currency 检查器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts", server.listAccount)
	router.DELETE("/accounts/:id", server.deleteAccount)
	router.PUT("/accounts", server.updateAccount)

	router.POST("/transfer", server.createTransfer)

	server.router = router
	return server
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) {
	server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
```



使用：

```go
type createAccountRequest struct {
	Owner    string `json:"owner" binding:"required"` // binding 标签表示 gin 验证必填选项
	Currency string `json:"currency" binding:"required,currency"`
}
```

## 密码hash加密
utils/password.go
```go
package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns the bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedPassword), nil
}

func CheckPassword(password string, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

测试：
utils/password_test.go
```go
package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password := RandomString(6)
	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)

	err = CheckPassword(password, hashedPassword)
	require.NoError(t, err)

	wrongPassword := RandomString(7)
	err = CheckPassword(wrongPassword, hashedPassword)
	require.EqualError(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())
}
```