package comfylite3

// import (
// 	"database/sql"
// 	"database/sql/driver"
// 	"io"
// )

// func convertValues(vals []driver.Value) []interface{} {
// 	result := make([]interface{}, len(vals))
// 	for i, v := range vals {
// 		result[i] = v
// 	}
// 	return result
// }

// type ComfyDriver struct {
// 	comfy *ComfyDB
// }

// func (cd *ComfyDriver) Open(name string) (driver.Conn, error) {
// 	return &comfyConn{comfy: cd.comfy}, nil
// }

// type comfyConn struct {
// 	comfy *ComfyDB
// }

// func (cc *comfyConn) Prepare(query string) (driver.Stmt, error) {
// 	return &comfyStmt{comfy: cc.comfy, query: query}, nil
// }

// func (cc *comfyConn) Close() error {
// 	return cc.comfy.Close()
// }

// func (cc *comfyConn) Begin() (driver.Tx, error) {
// 	return &comfyTx{comfy: cc.comfy}, nil
// }

// type comfyStmt struct {
// 	comfy *ComfyDB
// 	query string
// }

// func (cs *comfyStmt) Close() error {
// 	return nil
// }

// func (cs *comfyStmt) NumInput() int {
// 	return -1 // We don't know the number of placeholders
// }

// func (cs *comfyStmt) Exec(args []driver.Value) (driver.Result, error) {
// 	id := cs.comfy.New(func(db *sql.DB) (interface{}, error) {
// 		return db.Exec(cs.query, convertValues(args)...)
// 	})
// 	result := <-cs.comfy.WaitFor(id)
// 	if err, ok := result.(error); ok {
// 		return nil, err
// 	}
// 	return result.(sql.Result), nil
// }

// func (cs *comfyStmt) Query(args []driver.Value) (driver.Rows, error) {
// 	id := cs.comfy.New(func(db *sql.DB) (interface{}, error) {
// 		return db.Query(cs.query, convertValues(args)...)
// 	})
// 	result := <-cs.comfy.WaitFor(id)
// 	if err, ok := result.(error); ok {
// 		return nil, err
// 	}
// 	return &comfyRows{rows: result.(*sql.Rows)}, nil
// }

// type comfyRows struct {
// 	rows *sql.Rows
// }

// func (cr *comfyRows) Columns() []string {
// 	cols, _ := cr.rows.Columns()
// 	return cols
// }

// func (cr *comfyRows) Close() error {
// 	return cr.rows.Close()
// }

// func (cr *comfyRows) Next(dest []driver.Value) error {
// 	if !cr.rows.Next() {
// 		return io.EOF
// 	}
// 	columns, _ := cr.rows.Columns()
// 	values := make([]interface{}, len(columns))
// 	for i := range values {
// 		values[i] = new(interface{})
// 	}
// 	err := cr.rows.Scan(values...)
// 	if err != nil {
// 		return err
// 	}
// 	for i, v := range values {
// 		dest[i] = *(v.(*interface{}))
// 	}
// 	return nil
// }

// type comfyTx struct {
// 	comfy *ComfyDB
// 	tx    *sql.Tx
// }

// func (ct *comfyTx) Commit() error {
// 	id := ct.comfy.New(func(db *sql.DB) (interface{}, error) {
// 		return nil, ct.tx.Commit()
// 	})
// 	result := <-ct.comfy.WaitFor(id)
// 	if err, ok := result.(error); ok {
// 		return err
// 	}
// 	return nil
// }

// func (ct *comfyTx) Rollback() error {
// 	id := ct.comfy.New(func(db *sql.DB) (interface{}, error) {
// 		return nil, ct.tx.Rollback()
// 	})
// 	result := <-ct.comfy.WaitFor(id)
// 	if err, ok := result.(error); ok {
// 		return err
// 	}
// 	return nil
// }
