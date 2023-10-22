package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/shadiestgoat/log"
)

var pool *pgxpool.Pool

func OpenDB(dbURL string) {
	conf, err := pgxpool.ParseConfig(dbURL)
	log.FatalIfErr(err, "parsing db config")

	db, err := pgxpool.ConnectConfig(context.Background(), conf)
	log.FatalIfErr(err, "connecting to pool")

	err = db.Ping(context.Background())
	log.FatalIfErr(err, "pinging the db")

	pool = db

	for _, sql := range setup {
		_, err = Exec(sql[0])
		log.FatalIfErr(err, sql[1])
	}
}

func Exec(sql string, args ...any) (pgconn.CommandTag, error) {
	v1, err := pool.Exec(context.Background(), sql, args...)
	if err != nil {
		log.Error("Couldn't exec '%s': %v", sql, err)
	}
	return v1, err
}

func Query(sql string, args ...any) (pgx.Rows, error) {
	rows, err := pool.Query(context.Background(), sql, args...)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Error("Couldn't fetch '%s': %v", sql, err)
	}
	return rows, err
}

func QueryRow(sql string, args []any, scanTarget ...any) error {
	row := pool.QueryRow(context.Background(), sql, args...)
	err := row.Scan(scanTarget...)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Error("Couldn't row fetch '%s': %v", sql, err)
	}
	return err
}

// Query row with 1 condition
func QueryRowID(sql string, arg any, scanTarget ...any) error {
	row := pool.QueryRow(context.Background(), sql, arg)
	err := row.Scan(scanTarget...)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Error("Couldn't row fetch '%s': %v", sql, err)
	}
	return err
}

func Exists(table string, conditions string, values ...any) bool {
	var ret bool
	err := pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE %s)`, table, conditions), values...).Scan(&ret)
	if err != nil {
		log.Error("Couldn't row exist fetch from table '%s', conditions '%s': %v", table, conditions, err)
	}
	return ret
}

func Insert(table string, columns []string, values [][]any) (int64, error) {
	if len(values) == 0 {
		return 0, nil
	}

	n, err := pool.CopyFrom(context.Background(), pgx.Identifier{table}, columns, pgx.CopyFromRows(values))

	if err != nil {
		b, _ := json.MarshalIndent(values, "", "\t")
		log.Error("Couldn't insert into table '%s' (%v), Values:\n%s", table, columns, string(b))
	}

	return n, err
}

func InsertOne(table string, columns []string, values ...any) (int64, error) {
	return Insert(table, columns, [][]any{values})
}

func Close() {
	pool.Close()
}
