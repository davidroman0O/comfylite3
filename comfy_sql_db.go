package comfylite3

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"
)

/// It's time to replace my own version of sql.DB to be plug and play with other libraries

// implement Ping() error of sql.DB with Comfy
func (c *ComfyDB) Ping() error {
	pingID := c.New(func(db *sql.DB) (interface{}, error) {
		return nil, db.Ping()
	})
	result := <-c.WaitForChn(pingID)
	switch data := result.(type) {
	case error:
		return data
	default:
		return nil
	}
}

func (c *ComfyDB) Begin() (*sql.Tx, error) {
	txID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Begin()
	})
	result := <-c.WaitForChn(txID)
	switch data := result.(type) {
	case *sql.Tx:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	txID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.BeginTx(ctx, opts)
	})
	result := <-c.WaitForChn(txID)
	switch data := result.(type) {
	case *sql.Tx:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) Conn(ctx context.Context) (*sql.Conn, error) {
	connID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Conn(ctx)
	})
	result := <-c.WaitForChn(connID)
	switch data := result.(type) {
	case *sql.Conn:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) Driver() driver.Driver {
	driverID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Driver(), nil
	})
	result := <-c.WaitForChn(driverID)
	switch data := result.(type) {
	case driver.Driver:
		return data
	default:
		return nil
	}
}

func (c *ComfyDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	execID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec(query, args...)
	})
	result := <-c.WaitForChn(execID)
	switch data := result.(type) {
	case sql.Result:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	execID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.ExecContext(ctx, query, args...)
	})
	result := <-c.WaitForChn(execID)
	switch data := result.(type) {
	case sql.Result:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) PingContext(ctx context.Context) error {
	pingID := c.New(func(db *sql.DB) (interface{}, error) {
		return nil, db.PingContext(ctx)
	})
	result := <-c.WaitForChn(pingID)
	switch data := result.(type) {
	case error:
		return data
	default:
		return nil
	}
}

func (c *ComfyDB) Prepare(query string) (*sql.Stmt, error) {
	stmtID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Prepare(query)
	})
	result := <-c.WaitForChn(stmtID)
	switch data := result.(type) {
	case *sql.Stmt:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmtID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.PrepareContext(ctx, query)
	})
	result := <-c.WaitForChn(stmtID)
	switch data := result.(type) {
	case *sql.Stmt:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rowsID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Query(query, args...)
	})
	result := <-c.WaitForChn(rowsID)
	switch data := result.(type) {
	case *sql.Rows:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rowsID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.QueryContext(ctx, query, args...)
	})
	result := <-c.WaitForChn(rowsID)
	switch data := result.(type) {
	case *sql.Rows:
		return data, nil
	default:
		return nil, data.(error)
	}
}

func (c *ComfyDB) QueryRow(query string, args ...interface{}) *sql.Row {
	rowID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.QueryRow(query, args...), nil
	})
	result := <-c.WaitForChn(rowID)
	switch data := result.(type) {
	case *sql.Row:
		return data
	default:
		return nil
	}
}

func (c *ComfyDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	rowID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.QueryRowContext(ctx, query, args...), nil
	})
	result := <-c.WaitForChn(rowID)
	switch data := result.(type) {
	case *sql.Row:
		return data
	default:
		return nil
	}
}

func (c *ComfyDB) SetConnMaxIdleTime(d time.Duration) {
	c.New(func(db *sql.DB) (interface{}, error) {
		db.SetConnMaxIdleTime(d)
		return nil, nil
	})
}

func (c *ComfyDB) SetConnMaxLifetime(d time.Duration) {
	c.New(func(db *sql.DB) (interface{}, error) {
		db.SetConnMaxLifetime(d)
		return nil, nil
	})
}

func (c *ComfyDB) SetMaxIdleConns(n int) {
	c.New(func(db *sql.DB) (interface{}, error) {
		db.SetMaxIdleConns(n)
		return nil, nil
	})
}

func (c *ComfyDB) SetMaxOpenConns(n int) {
	c.New(func(db *sql.DB) (interface{}, error) {
		db.SetMaxOpenConns(n)
		return nil, nil
	})
}

func (c *ComfyDB) Stats() sql.DBStats {
	statsID := c.New(func(db *sql.DB) (interface{}, error) {
		return db.Stats(), nil
	})
	result := <-c.WaitForChn(statsID)
	switch data := result.(type) {
	case sql.DBStats:
		return data
	default:
		return sql.DBStats{}
	}
}
