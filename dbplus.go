package sqlplus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"log"
	"strings"
	"sync"
	"time"
)

type DbPlus struct {
	p      int32
	dbs    []*sql.DB
	l      sync.Mutex
	tracer trace.Tracer
}

func Open(driverName string, t trace.Tracer, dsns ...string) (dp *DbPlus, err error) {
	if len(dsns) < 1 || len(dsns) > 254 {
		err = fmt.Errorf("连接至少一个，或低于255个")
	}
	dp = &DbPlus{tracer: t}
	for _, dsn := range dsns {
		if db, err := sql.Open(driverName, dsn); err == nil {
			dp.dbs = append(dp.dbs, db)
		} else {
			return nil, err
		}
	}
	if len(dp.dbs) == 0 {
		err = fmt.Errorf("no db err")
	}
	return
}

func (db *DbPlus) handleError(err error) {
	if strings.Contains(err.Error(), "connection refused") {

	}
}

func (db *DbPlus) Writer() *sql.DB {
	return db.dbs[0]
}

func (db *DbPlus) detect(sql string) *sql.DB {
	if db == nil {
		log.Println("no sql connection")
		return nil
	}
	db.l.Lock()
	defer db.l.Unlock()
	if !strings.HasPrefix(strings.ToLower(sql), "select") &&
		!strings.HasPrefix(strings.ToLower(sql), "with") &&
		!strings.HasPrefix(strings.ToLower(sql), "refresh") {
		return db.dbs[0]
	} else if len(db.dbs) == 1 {
		return db.dbs[0]
	} else {
		db.p++
		if db.p == 0 || db.p >= int32(len(db.dbs)) {
			db.p = 1
		}
		return db.dbs[db.p]
	}
}

func (db *DbPlus) QueryStructContext(ctx context.Context, obj interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisStruct(obj)
	if err != nil {
		return
	}

	b.rows, err = db.QueryContext(ctx, b.mustLimit1(query), args...)
	if err != nil {
		return
	}
	defer b.rows.Close()

	err = b.parseStruct()
	if err != nil {
		return
	}

	return
}

func (db *DbPlus) QueryStruct(obj interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisStruct(obj)
	if err != nil {
		return
	}

	b.rows, err = db.Query(b.mustLimit1(query), args...)
	if err != nil {
		return
	}
	defer b.rows.Close()

	err = b.parseStruct()
	if err != nil {
		return
	}

	return
}

func (db *DbPlus) QuerySliceContext(ctx context.Context, list interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisSlice(list)
	if err != nil {
		return
	}

	b.rows, err = db.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}
	defer b.rows.Close()

	err = b.parseSlideAll()
	if err != nil {
		return
	}
	return
}

func (db *DbPlus) QuerySlice(list interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisSlice(list)
	if err != nil {
		return
	}

	b.rows, err = db.Query(query, args...)
	if err != nil {
		return
	}
	defer b.rows.Close()

	err = b.parseSlideAll()
	if err != nil {
		return
	}

	return
}

// Exists 判断记录是否存在
func (db *DbPlus) Exists(query string, args ...interface{}) (exists bool, err error) {
	if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(query)), "select") &&
		!strings.HasPrefix(strings.TrimSpace(strings.ToLower(query)), "with") &&
		strings.HasPrefix(strings.TrimSpace(strings.ToLower(query)), "refresh") {
		return false, errors.New("just support select query")
	}
	err = db.QueryRow(fmt.Sprintf("select exists (%s)", query), args...).Scan(&exists)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

// ExistsContext 判断记录是否存在
func (db *DbPlus) ExistsContext(c context.Context, query string, args ...interface{}) (exists bool, err error) {
	if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(query)), "select") &&
		!strings.HasPrefix(strings.TrimSpace(strings.ToLower(query)), "with") &&
		!strings.HasPrefix(strings.TrimSpace(strings.ToLower(query)), "refresh") {
		return false, errors.New("just support select query")
	}
	err = db.QueryRowContext(c, fmt.Sprintf("select exists (%s)", query), args...).Scan(&exists)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (db *DbPlus) Begin() (*TxPlus, error) {
	tx := &TxPlus{}
	var err error
	tx.Tx, err = db.dbs[0].Begin()
	return tx, err
}

func (db *DbPlus) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TxPlus, error) {
	tx := &TxPlus{}
	var err error
	tx.Tx, err = db.dbs[0].BeginTx(ctx, opts)
	return tx, err
}

func (db *DbPlus) Prepare(query string) (*sql.Stmt, error) {
	return db.PrepareContext(context.Background(), query)
}

func (db *DbPlus) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	_ctx, span := db.tracer.Start(ctx, "db.prepare")
	defer span.End()
	span.SetAttributes(attribute.String("query", query))
	stmt, err := db.detect(query).PrepareContext(_ctx, query)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return stmt, err
}

func (db *DbPlus) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

func (db *DbPlus) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	_ctx, span := db.tracer.Start(ctx, "db.exec")
	defer span.End()
	span.SetAttributes(attribute.String("query", query))
	res, err := db.detect(query).ExecContext(_ctx, query, args...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return res, err
}

func (db *DbPlus) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

func (db *DbPlus) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	_ctx, span := db.tracer.Start(ctx, "db.query")
	defer span.End()
	span.SetAttributes(attribute.String("query", query))
	res, err := db.detect(query).QueryContext(_ctx, query, args...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return res, err
}

func (db *DbPlus) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DbPlus) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	_ctx, span := db.tracer.Start(ctx, "db.query.row")
	defer span.End()
	span.SetAttributes(attribute.String("query", query))
	return db.detect(query).QueryRowContext(_ctx, query, args...)
}

func (db *DbPlus) SetMaxIdleConns(ns ...int) {
	for k, v := range db.dbs {
		if k < len(ns) {
			v.SetMaxIdleConns(ns[k])
		} else {
			v.SetMaxIdleConns(ns[len(ns)-1])
		}
	}
}

func (db *DbPlus) SetMaxOpenConns(ns ...int) {
	for k, v := range db.dbs {
		if k < len(ns) {
			v.SetMaxOpenConns(ns[k])
		} else {
			v.SetMaxOpenConns(ns[len(ns)-1])
		}
	}
}

func (db *DbPlus) SetConnMaxLifetime(ds ...time.Duration) {
	for k, v := range db.dbs {
		if k < len(ds) {
			v.SetConnMaxLifetime(ds[k])
		} else {
			v.SetConnMaxLifetime(ds[len(ds)-1])
		}
	}
}

func (db *DbPlus) Close() (err error) {
	for _, v := range db.dbs {
		err = v.Close()
	}
	return
}
