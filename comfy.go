package comfylite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davidroman0O/retrypool"
	_ "github.com/mattn/go-sqlite3"
)

// Callback provided by a developer to be executed when the scheduler is ready for it
type SqlFn func(db *sql.DB) (interface{}, error)

type workItem struct {
	id     uint64
	fn     SqlFn
	result chan interface{}
}

// Default Memory Connection
const memoryConn = "file::memory:?_mutex=full&cache=shared&_timeout=5000"

// Default File Connection
const fileConn = "file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000"

type onPanic func(v interface{}, stackTrace string)

type Migration struct {
	Version uint
	Label   string
	Up      func(tx *sql.Tx) error
	Down    func(tx *sql.Tx) error
}

// Create a new migration with a version, label, and up and down functions.
func NewMigration(version uint, label string, up, down func(tx *sql.Tx) error) Migration {
	return Migration{
		Version: version,
		Label:   label,
		Up:      up,
		Down:    down,
	}
}

// ComfyDB is a wrapper around sqlite3 that provides a simple API for executing SQL queries with goroutines.
type ComfyDB struct {
	db      *sql.DB
	count   atomic.Uint64
	results sync.Map

	migrations         []Migration
	migrationTableName string

	memory bool
	path   string
	conn   string

	pool        *retrypool.Pool[*workItem]
	poolOptions []retrypool.Option[*workItem]
}

type ComfyOption func(*ComfyDB)

// WithMigrationTableName sets the name of the migration table.
func WithMigrationTableName(name string) ComfyOption {
	return func(o *ComfyDB) {
		o.migrationTableName = name
	}
}

// WithPath sets the path of the database file.
func WithPath(path string) ComfyOption {
	return func(o *ComfyDB) {
		o.path = path
		o.memory = false
	}
}

// WithMemory sets the database to be in-memory.
func WithMemory() ComfyOption {
	return func(o *ComfyDB) {
		o.memory = true
	}
}

// WithConnection sets a custom connection string for the database.
func WithConnection(conn string) ComfyOption {
	return func(o *ComfyDB) {
		o.conn = conn
	}
}

// Records your migrations for your database.
func WithMigration(migrations ...Migration) ComfyOption {
	return func(c *ComfyDB) {
		c.migrations = append(c.migrations, migrations...)
	}
}

// WithRetryAttempts sets maximum retry attempts for failed operations
func WithRetryAttempts(attempts int) ComfyOption {
	return func(c *ComfyDB) {
		c.poolOptions = append(c.poolOptions, retrypool.WithAttempts[*workItem](attempts))
	}
}

// WithRetryDelay sets delay between retries
func WithRetryDelay(delay time.Duration) ComfyOption {
	return func(c *ComfyDB) {
		c.poolOptions = append(c.poolOptions, retrypool.WithDelay[*workItem](delay))
	}
}

// WithPanicHandler sets custom panic handler
func WithPanicHandler(handler onPanic) ComfyOption {
	return func(c *ComfyDB) {
		c.poolOptions = append(c.poolOptions, retrypool.WithPanicHandler[*workItem](func(task *workItem, v interface{}, stackTrace string) {
			handler(v, stackTrace)
		}))
	}
}

// Close the database connection.
func (c *ComfyDB) Close() error {
	// Close the retrypool
	if err := c.pool.Shutdown(); err != nil {
		if err != context.Canceled {
			return err
		}
	}

	// Close the database connection
	return c.db.Close()
}

// Prepare the eventual creation of the migration table.
func (c *ComfyDB) prepareMigration() error {
	newTableID := c.New(func(db *sql.DB) (interface{}, error) {
		_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %v (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version INTEGER UNIQUE NOT NULL,
			description VARCHAR(255) UNIQUE NOT NULL
		)`, c.migrationTableName))
		return nil, err
	})
	result, err := c.WaitFor(newTableID)
	if err != nil {
		return err
	}
	if errResult, ok := result.(error); ok {
		return errResult
	}
	return nil
}

// Sort the migrations by version.
func (c *ComfyDB) sort() []Migration {
	cp := make([]Migration, len(c.migrations))
	copy(cp, c.migrations)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].Version < cp[j].Version
	})
	return cp
}

// Create a new ComfyLite3 wrapper around sqlite3.
// Instantiate a scheduler to process your queries.
func New(opts ...ComfyOption) (*ComfyDB, error) {
	c := &ComfyDB{
		memory:             true,
		migrations:         []Migration{},
		migrationTableName: "_migrations",
		poolOptions:        make([]retrypool.Option[*workItem], 0),
	}

	c.count.Store(1)

	for _, opt := range opts {
		opt(c)
	}

	// Open the database connection
	var err error
	if c.conn != "" {
		c.db, err = sql.Open("sqlite3", c.conn)
	} else if c.memory {
		c.db, err = sql.Open("sqlite3", memoryConn)
	} else {
		if c.path == "" {
			return nil, fmt.Errorf("path is required")
		}
		c.db, err = sql.Open("sqlite3", fmt.Sprintf(fileConn, c.path))
	}

	if err != nil {
		return nil, err
	}

	c.db.SetMaxOpenConns(1)
	c.db.SetMaxIdleConns(1)

	// Initialize the retrypool with a single worker
	c.pool = retrypool.New[*workItem](
		context.Background(),
		[]retrypool.Worker[*workItem]{c},
		c.poolOptions...,
	)

	// Prepare migrations
	if err := c.prepareMigration(); err != nil {
		return nil, err
	}

	return c, nil
}

// Implement the Worker interface from retrypool
func (c *ComfyDB) Run(ctx context.Context, item *workItem) error {
	// Execute the function
	res, err := item.fn(c.db)

	// Store the result
	if err != nil {
		item.result <- err
	} else {
		item.result <- res
	}
	close(item.result)

	return nil
}

// New adds a new SQL function to be executed
func (c *ComfyDB) New(fn SqlFn) uint64 {

	// Check if we're about to overflow and reset if necessary
	if c.count.Load() == math.MaxUint64 {
		c.count.Store(1) // Reset to 1
	}

	item := &workItem{
		id:     c.count.Add(1),
		fn:     fn,
		result: make(chan interface{}, 1),
	}

	// Store the work item
	c.results.Store(item.id, item)

	// Dispatch the work item to the retrypool
	err := c.pool.Submit(item)
	if err != nil {
		// Handle the error appropriately
		// For now, let's panic
		panic(fmt.Sprintf("Failed to dispatch work item: %v", err))
	}

	return item.id
}

// WaitFor waits for the result of a workID (your query).
func (c *ComfyDB) WaitFor(workID uint64) (interface{}, error) {
	value, ok := c.results.Load(workID)
	if !ok {
		return nil, fmt.Errorf("workID not found")
	}
	item := value.(*workItem)

	// Wait for the result
	select {
	case res := <-item.result:
		// Delete the item from the results map after consuming the result
		c.results.Delete(workID)
		return res, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for result")
	}
}

// WaitForChn waits for the result of a workID (your query) and returns a channel.
func (c *ComfyDB) WaitForChn(workID uint64) <-chan interface{} {
	value, ok := c.results.Load(workID)
	if !ok {
		ch := make(chan interface{})
		close(ch)
		return ch
	}
	item := value.(*workItem)

	// Create a channel to return
	resultCh := make(chan interface{}, 1)

	go func() {
		res := <-item.result
		// Delete the item from the results map after consuming the result
		c.results.Delete(workID)
		resultCh <- res
		close(resultCh)
	}()

	return resultCh
}

// Migrate up all the available migrations.
func (c *ComfyDB) Up(ctx context.Context) error {
	if err := c.prepareMigration(); err != nil {
		return err
	}

	index, err := c.Index()
	if err != nil {
		return err
	}

	migrationExists := map[uint]bool{}
	for _, v := range index {
		migrationExists[v] = true
	}

	localSorted := c.sort()

	migrationUpID := c.New(func(db *sql.DB) (interface{}, error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		for _, migration := range localSorted {
			if migration.Version == 0 || migration.Label == "" {
				return nil, fmt.Errorf("invalid migration: version and label must be set")
			}

			if migration.Up == nil || migration.Down == nil {
				return nil, fmt.Errorf("invalid migration: up and down must be set")
			}

			if migrationExists[migration.Version] {
				continue
			}

			if err := migration.Up(tx); err != nil {
				return nil, err
			}

			if _, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %v (version, description) VALUES (?, ?)", c.migrationTableName), migration.Version, migration.Label); err != nil {
				return nil, fmt.Errorf("failed to insert migration (version=%v, description=%s): %w", migration.Version, migration.Label, err)
			}
		}
		return nil, tx.Commit()
	})
	result, err := c.WaitFor(migrationUpID)
	if err != nil {
		return err
	}
	if errResult, ok := result.(error); ok {
		return errResult
	}
	return nil
}

// Migrate down using the amount of iterations to rollback.
func (c *ComfyDB) Down(ctx context.Context, amount int) error {
	if err := c.prepareMigration(); err != nil {
		return err
	}

	index, err := c.Index()
	if err != nil {
		return err
	}

	if len(index) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	if amount > len(index) {
		amount = len(index)
	}

	migrationExists := map[uint]bool{}
	for _, v := range index {
		migrationExists[v] = true
	}

	localSorted := c.sort()

	migrationDownID := c.New(func(db *sql.DB) (interface{}, error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		for i := len(index) - 1; i >= len(index)-amount; i-- {
			migration := localSorted[index[i]-1]

			if migration.Version == 0 || migration.Label == "" {
				return nil, fmt.Errorf("invalid migration: version and label must be set")
			}

			if migration.Up == nil || migration.Down == nil {
				return nil, fmt.Errorf("invalid migration: up and down must be set")
			}

			if !migrationExists[migration.Version] {
				return nil, fmt.Errorf("migration (version=%v, label=%s) doesn't exist", migration.Version, migration.Label)
			}

			if err := migration.Down(tx); err != nil {
				return nil, err
			}

			if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %v WHERE version = ?", c.migrationTableName), migration.Version); err != nil {
				return nil, fmt.Errorf("failed to delete migration (version=%v, label=%s): %w", migration.Version, migration.Label, err)
			}
		}
		return nil, tx.Commit()
	})
	result, err := c.WaitFor(migrationDownID)
	if err != nil {
		return err
	}
	if errResult, ok := result.(error); ok {
		return errResult
	}
	return nil
}

// Get all versions of the migrations.
func (c *ComfyDB) Index() ([]uint, error) {
	currentIndexID := c.New(func(db *sql.DB) (interface{}, error) {
		var versions []uint
		rows, err := db.Query(fmt.Sprintf("SELECT version FROM %v ORDER BY version ASC", c.migrationTableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var version uint
			if err := rows.Scan(&version); err != nil {
				return nil, err
			}
			versions = append(versions, version)
		}
		return versions, nil
	})
	result, err := c.WaitFor(currentIndexID)
	if err != nil {
		return nil, err
	}
	switch value := result.(type) {
	case []uint:
		return value, nil
	case error:
		if value == sql.ErrNoRows {
			return []uint{}, nil
		}
		return nil, value
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

// Get all migrations.
func (c *ComfyDB) Migrations() ([]Migration, error) {
	migrationsID := c.New(func(db *sql.DB) (interface{}, error) {
		var migrations []Migration
		rows, err := db.Query(fmt.Sprintf("SELECT version, description FROM %v ORDER BY version ASC", c.migrationTableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var version uint
			var description string
			if err := rows.Scan(&version, &description); err != nil {
				return nil, err
			}
			migrations = append(migrations, Migration{
				Version: version,
				Label:   description,
			})
		}
		return migrations, nil
	})
	result, err := c.WaitFor(migrationsID)
	if err != nil {
		return nil, err
	}
	switch value := result.(type) {
	case []Migration:
		return value, nil
	case error:
		if value == sql.ErrNoRows {
			return []Migration{}, nil
		}
		return nil, value
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

// Get current version of the migrations.
func (c *ComfyDB) Version() (uint, error) {
	versionID := c.New(func(db *sql.DB) (interface{}, error) {
		var version uint
		row := db.QueryRow(fmt.Sprintf("SELECT version FROM %v ORDER BY version DESC LIMIT 1", c.migrationTableName))
		err := row.Scan(&version)
		if err != nil {
			if err == sql.ErrNoRows {
				return uint(0), nil
			}
			return nil, err
		}
		return version, nil
	})
	result, err := c.WaitFor(versionID)
	if err != nil {
		return 0, err
	}
	switch value := result.(type) {
	case uint:
		return value, nil
	case error:
		return 0, value
	default:
		return 0, fmt.Errorf("unexpected type")
	}
}

// Properties of one column in a table.
// Columns: cid name type notnull dflt_value pk
type Column struct {
	CID       int
	Name      string
	Type      string
	NotNull   bool
	DfltValue *string
	Pk        bool
}

// Show all tables in the database.
// Returns a slice of the names of the tables.
func (c *ComfyDB) ShowTables() ([]string, error) {
	tablesID := c.New(func(db *sql.DB) (interface{}, error) {
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}
		return tables, nil
	})
	result, err := c.WaitFor(tablesID)
	if err != nil {
		return nil, err
	}
	switch value := result.(type) {
	case []string:
		return value, nil
	case error:
		return nil, value
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

// Show all columns in a table.
func (c *ComfyDB) ShowColumns(table string) ([]Column, error) {
	columnsID := c.New(func(db *sql.DB) (interface{}, error) {
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info('%v')", table))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var cols []Column
		for rows.Next() {
			var col Column
			// cid name type notnull dflt_value pk
			if err := rows.Scan(&col.CID, &col.Name, &col.Type, &col.NotNull, &col.DfltValue, &col.Pk); err != nil {
				return nil, err
			}
			cols = append(cols, col)
		}
		return cols, nil
	})
	result, err := c.WaitFor(columnsID)
	if err != nil {
		return nil, err
	}
	switch value := result.(type) {
	case []Column:
		return value, nil
	case error:
		return nil, value
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

// RunSQL allows executing a custom SQL function and waits for its result.
func (c *ComfyDB) RunSQL(fn SqlFn) (interface{}, error) {
	workID := c.New(fn)
	return c.WaitFor(workID)
}

func (c *ComfyDB) IsLocked() (bool, error) {
	// Create a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try to execute a lightweight query that requires a write lock
	lockCheckID := c.New(func(db *sql.DB) (interface{}, error) {
		// Begin a transaction
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return true, nil // If we can't begin a transaction, assume it's locked
		}
		defer tx.Rollback()

		// Try to create and immediately drop a temporary table
		_, err = tx.ExecContext(ctx, `
            CREATE TEMPORARY TABLE IF NOT EXISTS _lock_check (id INTEGER PRIMARY KEY);
            DROP TABLE IF EXISTS _lock_check;
        `)

		if err != nil {
			// Check if the error is a database locked error
			if errors.Is(err, sql.ErrConnDone) ||
				errors.Is(err, sql.ErrTxDone) ||
				strings.Contains(err.Error(), "database is locked") {
				return true, nil
			}
			return false, err
		}

		return false, nil
	})

	result, err := c.WaitFor(lockCheckID)
	if err != nil {
		return false, err
	}

	switch v := result.(type) {
	case bool:
		return v, nil
	case error:
		return false, v
	default:
		return false, errors.New("unexpected result type from lock check")
	}
}

// WaitUnlocked waits for the database to become unlocked or until context is cancelled
func (c *ComfyDB) WaitUnlocked(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			locked, err := c.IsLocked()
			if err != nil {
				return err
			}
			if !locked {
				return nil
			}
		}
	}
}
