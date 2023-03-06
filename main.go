package main

import (
	"database/sql"
	"log"
	"simplebank/api"
	db "simplebank/db/sqlc"
	"simplebank/utils"

	_ "github.com/lib/pq"
)

func main() {
	conf, err := utils.LoadConig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	conn, err := sql.Open(conf.DBDriver, conf.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server := api.NewServer(store)

	server.Start(conf.ServerAddress)
}
