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

# 生成 CRUD 代码
sqlc:
	sqlc generate

# 显示日志、显示测试覆盖率、所有包执行测试单元
test:
	go test -v -cover ./...

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
