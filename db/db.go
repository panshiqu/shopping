package db

import (
	"database/sql"
	"log"
)

const dataSource = "root@tcp(localhost:3306)/shopping?charset=utf8mb4"

// Ins 实例
var Ins *sql.DB

func init() {
	var err error

	if Ins, err = sql.Open("mysql", dataSource); err != nil {
		log.Fatal(err)
	}

	if err = Ins.Ping(); err != nil {
		log.Fatal(err)
	}
}
