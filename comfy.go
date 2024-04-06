package comfylite3

import (
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

	memory bool
	path   string
	conn   string
}

type ComfyOption func(*ComfyDB)

func WithPath(path string) ComfyOption {
	return func(o *ComfyDB) {
		o.path = path
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

func (c *ComfyDB) Close() {
	c.shutdown <- struct{}{}
	close(c.shutdown)
	close(c.errors)
	c.db.Close()
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
					slog.Error("Error executing query", err)
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
