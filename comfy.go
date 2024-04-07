package comfylite3

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TODO @droman: add logger

var dbCount atomic.Uint64

func init() {
	dbCount.Store(0)
}

type SqlFn func(db *sql.DB) (interface{}, error)

type workItem struct {
	id uint64
	fn SqlFn
}

// Default Memory Connection
const memoryConn = "file::memory:?_mutex=full&cache=shared&_timeout=5000"

// Default File Connection
const fileConn = "file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000"

type Migration struct {
	Version uint
	Label   string
	Up      func(tx *sql.Tx) error
	Down    func(tx *sql.Tx) error
}

func NewMigration(version uint, label string, up, down func(tx *sql.Tx) error) Migration {
	return Migration{
		Version: version,
		Label:   label,
		Up:      up,
		Down:    down,
	}
}

type ComfyDB struct {
	id         uint64
	db         *sql.DB
	ringBuffer *RingBuffer[workItem]
	safeBuffer *safeMap[uint64, bool]
	safeChan   *safeMap[uint64, interface{}]
	shutdown   chan struct{}
	errors     chan error
	ticker     *time.Ticker
	count      atomic.Uint64

	migrations         []Migration
	migrationTableName string

	memory bool
	path   string
	conn   string
}

type ComfyOption func(*ComfyDB)

func WithMigrationTableName(name string) ComfyOption {
	return func(o *ComfyDB) {
		o.migrationTableName = name
	}
}

func WithPath(path string) ComfyOption {
	return func(o *ComfyDB) {
		o.path = path
		o.memory = false
	}
}

func WithMemory() ComfyOption {
	return func(o *ComfyDB) {
		o.memory = true
	}
}

func WithConnection(conn string) ComfyOption {
	return func(o *ComfyDB) {
		o.conn = conn
	}
}

func WithBuffer(size int64) ComfyOption {
	return func(c *ComfyDB) {
		c.ringBuffer = Buffer[workItem](size)
	}
}

func WithMigration(migrations ...Migration) ComfyOption {
	return func(c *ComfyDB) {
		// I think it's pretty comfy to have in your code all your migrations are a dummy array
		c.migrations = append(c.migrations, migrations...)
	}
}

func (c *ComfyDB) Close() {
	c.shutdown <- struct{}{}
	close(c.shutdown)
	close(c.errors)
	c.db.Close()
}

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
	result := <-c.WaitFor(newTableID)
	switch value := result.(type) {
	case error:
		return value
	default:
		return nil
	}
}

func (c *ComfyDB) sort() []Migration {
	cp := []Migration{}
	cp = append(cp, c.migrations...)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].Version < cp[j].Version
	})
	return cp
}

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
	result := <-c.WaitFor(tablesID)
	switch value := result.(type) {
	case error:
		return nil, value
	case []string:
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

// cid name type notnull dflt_value pk
type Column struct {
	CID       int
	Name      string
	Type      string
	NotNull   bool
	DfltValue *string
	Pk        bool
}

func (c *ComfyDB) ShowColumns(table string) ([]Column, error) {
	tablesID := c.New(func(db *sql.DB) (interface{}, error) {
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
	result := <-c.WaitFor(tablesID)
	switch value := result.(type) {
	case error:
		return nil, value
	case []Column:
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

func (c *ComfyDB) Up(ctx context.Context) error {
	var err error
	if err = c.prepareMigration(); err != nil {
		return err
	}

	var index []uint
	if index, err = c.Index(); err != nil {
		return err
	}

	migrationExists := map[uint]bool{}

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
				return nil, fmt.Errorf("duplicate migration: (version=%v, label=%s) already exists", migration.Version, migration.Label)
			}
			migrationExists[migration.Version] = true

			if slices.Contains(index, migration.Version) {
				// fmt.Printf("skipping migration: (version=%v, label=%s) already exists\n", migration.Version, migration.Label)
				continue
			}

			if err := migration.Up(tx); err != nil {
				return nil, err
			}

			if _, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %v (version, description) VALUES (?, ?)", c.migrationTableName), migration.Version, migration.Label); err != nil {
				return nil, fmt.Errorf("failed to insert migration (version=%v, description=%s): %w", migration.Version, migration.Label, err)
			}

			// fmt.Printf("migrated database up (version=%v, label=%s)\n", migration.Version, migration.Label)
		}
		return nil, tx.Commit()
	})
	result := <-c.WaitFor(migrationUpID)
	switch value := result.(type) {
	case error:
		return value
	default:
		return nil
	}
}

func (c *ComfyDB) Down(ctx context.Context, amount int) error {

	var err error
	if err = c.prepareMigration(); err != nil {
		return err
	}

	var index []uint
	if index, err = c.Index(); err != nil {
		return err
	}

	if len(index) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	if amount > len(index) {
		amount = len(index)
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

			if !slices.Contains(index, migration.Version) {
				return nil, fmt.Errorf("migration (version=%v, label=%s) doesn't exists", migration.Version, migration.Label)
			}

			if err := migration.Down(tx); err != nil {
				return nil, err
			}

			if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %v WHERE version = ?", c.migrationTableName), migration.Version); err != nil {
				return nil, fmt.Errorf("failed to insert migration (version=%v, label=%s): %w", migration.Version, migration.Label, err)
			}

			// fmt.Printf("migrated database down (version=%v, label=%s)\n", migration.Version, migration.Label)
		}

		return nil, tx.Commit()
	})
	result := <-c.WaitFor(migrationDownID)
	switch value := result.(type) {
	case error:
		return value
	default:
		return nil
	}
}

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
	result := <-c.WaitFor(migrationsID)
	switch value := result.(type) {
	case error:
		if value == sql.ErrNoRows {
			return []Migration{}, nil
		}
		return nil, value
	case []Migration:
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

func (c *ComfyDB) Version() (uint, error) {
	versionID := c.New(func(db *sql.DB) (interface{}, error) {
		var version uint
		row := db.QueryRow(fmt.Sprintf("SELECT version FROM %v ORDER BY version DESC LIMIT 1", c.migrationTableName))
		err := row.Scan(&version)
		return version, err
	})
	result := <-c.WaitFor(versionID)
	switch value := result.(type) {
	case error:
		if value == sql.ErrNoRows {
			return 0, nil
		}
		return 0, result.(error)
	case uint:
		return value, nil
	default:
		return 0, fmt.Errorf("unexpected type")
	}
}

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
	result := <-c.WaitFor(currentIndexID)
	switch value := result.(type) {
	case error:
		if value == sql.ErrNoRows {
			return []uint{}, nil
		}
		return nil, value
	case []uint:
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected type")
	}
}

func Comfy(opts ...ComfyOption) (*ComfyDB, error) {
	c := &ComfyDB{
		db:         nil,
		ringBuffer: Buffer[workItem](1024),
		safeBuffer: &safeMap[uint64, bool]{m: make(map[uint64]bool)},
		safeChan:   &safeMap[uint64, interface{}]{m: make(map[uint64]interface{})},
		shutdown:   make(chan struct{}),
		errors:     make(chan error),
		ticker:     time.NewTicker(1 * time.Microsecond),
		memory:     true,

		migrations:         []Migration{},
		migrationTableName: "_migrations",
	}

	c.count.Store(1)

	for _, opt := range opts {
		opt(c)
	}

	c.id = dbCount.Add(1)

	go func(instance *ComfyDB) {
		var err error
		var conn string
		if instance.conn != "" {
			conn = instance.conn
		} else {
			if instance.memory {
				conn = memoryConn
			} else {
				conn = fileConn
			}
		}

		if instance.memory {
			instance.db, err = sql.Open("sqlite3", conn)
			if err != nil {
				instance.errors <- err
				return
			}
		} else {
			if instance.path == "" {
				instance.errors <- fmt.Errorf("path is required")
				return
			}
			instance.db, err = sql.Open("sqlite3", fmt.Sprintf(conn, instance.path))
			if err != nil {
				instance.errors <- err
				return
			}
		}

		instance.errors <- nil

		instance.db.SetMaxOpenConns(1)
		instance.db.SetMaxIdleConns(1)

		for {
			select {
			case <-instance.ticker.C:
				if instance.ringBuffer.Len() == 0 {
					// todo @droman: add a counter to alternate the execution
					runtime.Gosched()
					time.Sleep(10 * time.Microsecond)
					continue
				}
				cb, ok := instance.ringBuffer.Pop()
				if !ok {
					continue
				}
				res, err := cb.fn(instance.db)
				if err != nil {
					instance.safeChan.Set(cb.id, err)
				} else {
					instance.safeChan.Set(cb.id, res)
				}
				instance.safeBuffer.Set(cb.id, true)
			case <-instance.shutdown:
				instance.ticker.Stop()
				return
			}
		}
	}(c)

	err := <-c.errors
	if err != nil {
		return nil, err
	}

	if err := c.prepareMigration(); err != nil {
		return nil, err
	}

	return c, nil
}

type safeMap[T comparable, V any] struct {
	m map[T]V
	sync.Mutex
}

func (sm *safeMap[T, V]) Set(k T, v V) {
	sm.Lock()
	sm.m[k] = v
	sm.Unlock()
}

func (sm *safeMap[T, V]) Get(k T) (V, bool) {
	sm.Lock()
	v, ok := sm.m[k]
	sm.Unlock()
	return v, ok
}

func (sm *safeMap[T, V]) Delete(k T) {
	sm.Lock()
	delete(sm.m, k)
	sm.Unlock()
}

func (c *ComfyDB) New(fn SqlFn) uint64 {
	item := workItem{
		id: c.count.Add(1),
		fn: fn,
	}
	c.ringBuffer.Push(item)
	c.safeBuffer.Set(item.id, false)
	return item.id
}

func (c *ComfyDB) Clear(id uint64) {
	c.safeBuffer.Delete(id)
	c.safeChan.Delete(id)
}

func (c *ComfyDB) IsDone(workID uint64) bool {
	v, ok := c.safeBuffer.Get(workID)
	if !ok {
		return false
	}
	return v
}

func (c *ComfyDB) WaitFor(workID uint64) <-chan interface{} {
	var cn interface{}
	var fine bool
	var future chan interface{} = make(chan interface{})
	loopTicker := time.NewTicker(10 * time.Microsecond)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-loopTicker.C:
				value, ok := c.safeBuffer.Get(workID)
				if !ok {
					runtime.Gosched()
					time.Sleep(10 * time.Microsecond)
					continue
				}
				if value {
					cn, fine = c.safeChan.Get(workID)
					if fine {
						loopTicker.Stop()
					}
					future <- cn
					done <- true
					close(done)
				}
			}
		}
	}()
	return future
}
