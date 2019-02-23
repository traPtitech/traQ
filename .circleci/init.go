//+build tools

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	conn, err := sql.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true",
		getEnvOrDefault("TEST_DB_USER", "root"),
		getEnvOrDefault("TEST_DB_PASSWORD", "password"),
		getEnvOrDefault("TEST_DB_HOST", "127.0.0.1"),
		getEnvOrDefault("TEST_DB_PORT", "3306"),
	))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	dbs := []string{
		"traq-test-repo-common",
		"traq-test-repo-ex1",
		"traq-test-repo-ex2",
	}
	for _, name := range dbs {
		if _, err = conn.Exec("CREATE DATABASE `" + name + "` CHARACTER SET = utf8mb4"); err != nil {
			panic(err)
		}
		log.Println("Database `" + name + "` was created")
	}
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}
