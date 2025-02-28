package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/godror/godror"
	"os"
	"strconv"
	"time"
)

var connectString = os.Getenv("DATABASE_CONNECTION_STRING")
var user = os.Getenv("DATABASE_USERNAME")
var password = os.Getenv("DATABASE_PASSWORD")
var threads = os.Getenv("QUERY_THREADS")
var queryString = "SELECT BANNER, BANNER_FULL FROM v$version"

func main() {
	db := getDB()
	numThreads, err := strconv.Atoi(threads)
	if err != nil {
		fmt.Println("QUERY_THREADS environment variable must be an int.")
		os.Exit(1)
	}

	fmt.Printf("Starting %d query threads\n", numThreads)
	ch := make(chan error, numThreads)
	for i := 0; i < numThreads; i++ {
		go query(ch, db, queryString)
	}

	hasErr := false
	for i := 0; i < numThreads; i++ {
		queryErr := <-ch
		if queryErr != nil {
			fmt.Printf("Query error from thread %d: %v\n", i, queryErr)
			hasErr = true
		}
	}
	close(ch)
	if !hasErr {
		fmt.Println("All query threads completed without error")
	}
	fmt.Println("Done!")
}

func getDB() *sql.DB {
	var P godror.ConnectionParams
	P.ConnectString = connectString
	P.Username = user
	P.Password = godror.NewPassword(password)
	P.ExternalAuth = sql.NullBool{
		Bool:  false,
		Valid: true,
	}

	P.PoolParams.WaitTimeout = time.Second * 5
	db := sql.OpenDB(godror.NewConnector(P))
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(0)
	return db
}

func query(ch chan error, db *sql.DB, q string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.QueryContext(ctx, q)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		ch <- errors.New("query timeout")
	}
	ch <- err
}
