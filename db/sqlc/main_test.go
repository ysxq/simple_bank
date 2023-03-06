package db

import (
	"database/sql"
	"log"
	"os"
	"simplebank/utils"
	"testing"

	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDb *sql.DB

func TestMain(m *testing.M) {
	conf, err := utils.LoadConig("../../")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDb, err = sql.Open(conf.DBDriver, conf.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDb)

	os.Exit(m.Run())
}
