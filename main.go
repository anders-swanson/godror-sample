package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/godror/godror"
	"log"
	"os"
	"strconv"
	"time"
)

var connectString = os.Getenv("DATABASE_CONNECTION_STRING")
var user = os.Getenv("DATABASE_USERNAME")
var password = os.Getenv("DATABASE_PASSWORD")
var threads = os.Getenv("QUERY_THREADS")
var usePooling = os.Getenv("USE_POOLING")
var queryString = `
SELECT
    dt.tablespace_name as tablespace,
    dt.contents as type,
    dt.block_size * dtum.used_space as bytes,
    dt.block_size * dtum.tablespace_size as max_bytes,
    dt.block_size * (dtum.tablespace_size - dtum.used_space) as free,
    dtum.used_percent
FROM  dba_tablespace_usage_metrics dtum, dba_tablespaces dt
WHERE dtum.tablespace_name = dt.tablespace_name
and dt.contents != 'TEMPORARY'
union
SELECT
    dt.tablespace_name as tablespace,
    'TEMPORARY' as type,
    dt.tablespace_size - dt.free_space as bytes,
    dt.tablespace_size as max_bytes,
    dt.free_space as free,
    ((dt.tablespace_size - dt.free_space) / dt.tablespace_size)
FROM  dba_temp_free_space dt
order by tablespace
`

func main() {
	db := getDB()
	numThreads, err := strconv.Atoi(threads)
	if err != nil {
		fmt.Println("QUERY_THREADS environment variable must be an int.")
		os.Exit(1)
	}

	for z := 0; ; z++ {
		fmt.Printf("Starting %d query threads\n", numThreads)
		ch := make(chan error, numThreads)
		for i := 0; i < numThreads; i++ {
			go query(ch, db, queryString)
		}

		for i := 0; i < numThreads; i++ {
			queryErr := <-ch
			if queryErr != nil {
				fmt.Printf("Query error from thread %d: %v\n", i, queryErr)
				os.Exit(1)
			}
		}
		close(ch)
		fmt.Printf("iteration %d\n", z)
	}
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

	if ok, _ := strconv.ParseBool(usePooling); ok {
		fmt.Println("Using OCI Connection Pooling")
		P.PoolParams.WaitTimeout = time.Second * 5
		P.PoolParams.SessionIncrement = 3
		P.PoolParams.MaxSessions = 100
		P.PoolParams.MinSessions = 10
		P.StandaloneConnection = sql.NullBool{
			Bool:  false,
			Valid: true,
		}
	}

	db := sql.OpenDB(godror.NewConnector(P))
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
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
