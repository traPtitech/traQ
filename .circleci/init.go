package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	conn, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=true")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for _, name := range []string{"traq-test-model", "traq-test-router"} {
		if _, err = conn.Exec("CREATE DATABASE `" + name + "` CHARACTER SET = utf8mb4"); err != nil {
			panic(err)
		}
		log.Println("Database `" + name + "` was created")
	}
}
