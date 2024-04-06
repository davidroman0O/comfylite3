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

const memoryConn = "file::memory:?_mutex=full&cache=shared&_timeout=5000"
const fileConn = "file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000" // ?_journal=WAL&_timeout=5000&_fk=true

type opts struct {
	memory bool
	path   string
}

var shutdown chan error

type Option func(*opts)

func WithPath(path string) Option {
	return func(o *opts) {
		o.path = path
	}
}

func WithMemory() Option {
	return func(o *opts) {
		o.memory = true
	}
}

func Initialize(options ...Option) error {
	o := &opts{}
	for _, opt := range options {
		opt(o)
	}

	shutdown = make(chan error)
	count.Add(1)              // should not be zero
	go scheduler(o, shutdown) // i will tell you when i'm dead

	return nil
}

// should change values later
var ticker = time.NewTicker(10 * time.Microsecond)

func Close() {
	shutdown <- nil
	close(shutdown)
}

type workItem struct {
	id uint64
	fn fn
}

var count atomic.Uint64

func New(fn fn) uint64 {
	item := workItem{
		id: count.Add(1),
		fn: fn,
	}
	ringBuffer.Push(item)
	safeBuffer.Set(item.id, false)
	return item.id
}

func IsDone(id uint64) bool {
	value, ok := safeBuffer.Get(id)
	if !ok {
		// if you got a uint64, that might not exists, you might be lying to me
		// so i'm lying to you too
		return false
	}
	return value
}

func WaitFor(id uint64) <-chan interface{} {
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
				value, ok := safeBuffer.Get(id)
				if !ok {
					runtime.Gosched()
					time.Sleep(10 * time.Microsecond)
					continue
				}
				if value {
					cn, fine = safeChan.Get(id)
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

var ringBuffer *RingBuffer[workItem] = Buffer[workItem](1024)
var safeBuffer *safeMap[uint64, bool] = &safeMap[uint64, bool]{m: make(map[uint64]bool)}
var safeChan *safeMap[uint64, interface{}] = &safeMap[uint64, interface{}]{m: make(map[uint64]interface{})}

type fn func(db *sql.DB) (interface{}, error)

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

func scheduler(o *opts, cerr chan error) {
	var db *sql.DB
	var err error

	if o.memory {
		db, err = sql.Open("sqlite3", memoryConn)
		if err != nil {
			cerr <- err
			return
		}
	} else {
		if o.path == "" {
			cerr <- fmt.Errorf("path is required")
			return
		}
		db, err = sql.Open("sqlite3", fmt.Sprintf(fileConn, o.path))
		if err != nil {
			cerr <- err
			return
		}
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for {
		select {
		case <-ticker.C:
			if ringBuffer.Len() == 0 {
				// todo @droman: add a counter to alternate the execution
				runtime.Gosched()
				time.Sleep(10 * time.Microsecond)
				continue
			}
			cb, ok := ringBuffer.Pop()
			if !ok {
				continue
			}
			res, err := cb.fn(db)
			if err != nil {
				slog.Error("Error executing query", err)
				safeChan.Set(cb.id, err)
			} else {
				safeChan.Set(cb.id, res)
			}
			safeBuffer.Set(cb.id, true)
		case <-shutdown:
			ticker.Stop()
			db.Close()
			return
		}
	}
}
