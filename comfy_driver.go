package comfylite3

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
)

type ComfyDriver struct {
	comfy   *ComfyDB
	connStr string
}

func (cd *ComfyDriver) Open(name string) (driver.Conn, error) {
	return &comfyConn{comfy: cd.comfy, connStr: cd.connStr}, nil
}

func (cd *ComfyDriver) Connect(ctx context.Context) (driver.Conn, error) {
	return cd.Open("")
}

func (cd *ComfyDriver) Driver() driver.Driver {
	return cd
}

type comfyConn struct {
	comfy   *ComfyDB
	connStr string
}

func (cc *comfyConn) Prepare(query string) (driver.Stmt, error) {
	return &comfyStmt{comfy: cc.comfy, query: query}, nil
}

func (cc *comfyConn) Close() error {
	return nil
}

func (cc *comfyConn) Begin() (driver.Tx, error) {
	return &comfyTx{comfy: cc.comfy}, nil
}

type comfyStmt struct {
	comfy *ComfyDB
	query string
}

func (cs *comfyStmt) Close() error {
	return nil
}

func (cs *comfyStmt) NumInput() int {
	return -1
}

func (cs *comfyStmt) Exec(args []driver.Value) (driver.Result, error) {
	id := cs.comfy.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec(cs.query, convertValues(args)...)
	})
	result := <-cs.comfy.WaitFor(id)
	if err, ok := result.(error); ok {
		return nil, err
	}
	return result.(sql.Result), nil
}

func (cs *comfyStmt) Query(args []driver.Value) (driver.Rows, error) {
	id := cs.comfy.New(func(db *sql.DB) (interface{}, error) {
		return db.Query(cs.query, convertValues(args)...)
	})
	result := <-cs.comfy.WaitFor(id)
	if err, ok := result.(error); ok {
		return nil, err
	}
	return &comfyRows{rows: result.(*sql.Rows)}, nil
}

type comfyRows struct {
	rows *sql.Rows
}

func (cr *comfyRows) Columns() []string {
	cols, _ := cr.rows.Columns()
	return cols
}

func (cr *comfyRows) Close() error {
	return cr.rows.Close()
}

func (cr *comfyRows) Next(dest []driver.Value) error {
	if !cr.rows.Next() {
		return io.EOF
	}

	// Convert []driver.Value to []any
	args := make([]any, len(dest))
	for i, v := range dest {
		args[i] = &v
	}

	if err := cr.rows.Scan(args...); err != nil {
		return err
	}

	// Copy scanned values back to dest
	for i, v := range args {
		dest[i] = *v.(*driver.Value)
	}

	return nil
}

type comfyTx struct {
	comfy *ComfyDB
}

func (ct *comfyTx) Commit() error {
	return nil
}

func (ct *comfyTx) Rollback() error {
	return nil
}

func convertValues(vals []driver.Value) []interface{} {
	result := make([]interface{}, len(vals))
	for i, v := range vals {
		result[i] = v
	}
	return result
}

// OpenDB creates a new sql.DB instance using ComfyDB
func OpenDB(comfy *ComfyDB, options ...string) *sql.DB {
	connStr := comfy.conn

	// If comfy.conn is empty, use the default connection string
	if connStr == "" {
		if comfy.memory {
			connStr = "file::memory:"
		} else {
			connStr = fmt.Sprintf("file:%s", comfy.path)
		}
	}

	// Parse existing options
	existingOptions := make(map[string]bool)
	if strings.Contains(connStr, "?") {
		parts := strings.SplitN(connStr, "?", 2)
		connStr = parts[0]
		for _, opt := range strings.Split(parts[1], "&") {
			key := strings.SplitN(opt, "=", 2)[0]
			existingOptions[key] = true
		}
	}

	// Add new options
	newOptions := []string{}
	for _, opt := range options {
		key := strings.SplitN(opt, "=", 2)[0]
		if !existingOptions[key] {
			newOptions = append(newOptions, opt)
			existingOptions[key] = true
		}
	}

	// Append new options to connection string
	if len(newOptions) > 0 {
		if strings.Contains(connStr, "?") {
			connStr += "&"
		} else {
			connStr += "?"
		}
		connStr += strings.Join(newOptions, "&")
	}

	fmt.Printf("Connection string: %s\n", connStr) // Debug print

	db := sql.OpenDB(&ComfyDriver{
		comfy:   comfy,
		connStr: connStr,
	})

	// Explicitly enable foreign keys
	_, err := db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		fmt.Printf("Error setting foreign_keys pragma: %v\n", err)
	}

	return db
}
