package comfylite3

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMemory(t *testing.T) {

	comfyMe, err := New(
		WithMemory(),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer comfyMe.Close()

	chnCreate := make(chan uint64)
	go func() {
		chnCreate <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
			return db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
		})
	}()
	createID := <-chnCreate
	<-comfyMe.WaitForChn(createID)

	go func() {
		chnInsert := make(chan uint64)
		go func() {
			chnInsert <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
				return db.Exec("INSERT INTO users (name) VALUES (?)", "Jane Smith")
			})
		}()
		insertID := <-chnInsert
		<-comfyMe.WaitForChn(insertID)
	}()

	chnInsertDoe := make(chan uint64)

	go func() {
		chnInsertDoe <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
			return db.Exec("INSERT INTO users (name) VALUES (?)", "Doe Smith")
		})
	}()
	insertDoeID := <-chnInsertDoe

	chnInsertMain := comfyMe.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec("INSERT INTO users (name) VALUES (?)", "John Doe")
	})
	<-comfyMe.WaitForChn(chnInsertMain)
	<-comfyMe.WaitForChn(insertDoeID)

	chnSelect := make(chan uint64)
	go func() {
		chnSelect <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
			names := []string{}
			rows, err := db.Query("SELECT name FROM users")
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var name string
			for rows.Next() {
				err := rows.Scan(&name)
				if err != nil {
					return nil, err
				}
				names = append(names, name)
			}
			return names, nil
		})
	}()

	selectMainID := comfyMe.New(func(db *sql.DB) (interface{}, error) {
		names := []string{}
		rows, err := db.Query("SELECT name FROM users")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var name string
		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, name)
		}
		return names, nil
	})
	resultMainUsers := <-comfyMe.WaitForChn(selectMainID)
	selectGoID := <-chnSelect // almost same time, see if we got our select from the previous goroutine
	resultFromGo := <-comfyMe.WaitForChn(selectGoID)
	var names []string

	switch dd := resultMainUsers.(type) {
	case error:
		t.Fatal(resultMainUsers)
	case []string:
		names = dd
		fmt.Println(names)
	default:
		t.Fatal("unexpected result")
	}
	slog.Info("Data read")
	var goNames []string

	switch dd := resultFromGo.(type) {
	case error:
		t.Fatal(resultFromGo)
	case []string:
		goNames = dd
		fmt.Println(goNames)
	default:
		t.Fatal("unexpected result")
	}
	slog.Info("Data read")

	// Compare names and goNames
	if len(names) != len(goNames) {
		t.Fatal("Data mismatch")
	}

	for i := 0; i < len(names); i++ {
		if names[i] != goNames[i] {
			t.Fatal("Data mismatch")
		}
	}

}

func deleteTestDbFile() error {
	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "test.db") {
			err := os.Remove(file.Name())
			if err != nil {
				return nil
			}
		}
	}

	return nil
}

func TestFile(t *testing.T) {

	if err := deleteTestDbFile(); err != nil {
		t.Fatal(err)
	}

	comfyMe, err := New(
		WithPath("test.db"),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer comfyMe.Close()

	chnCreate := make(chan uint64)
	go func() {
		chnCreate <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
			return db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
		})
	}()
	createID := <-chnCreate
	<-comfyMe.WaitForChn(createID)

	go func() {
		chnInsert := make(chan uint64)
		go func() {
			chnInsert <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
				return db.Exec("INSERT INTO users (name) VALUES (?)", "Jane Smith")
			})
		}()
		insertID := <-chnInsert
		<-comfyMe.WaitForChn(insertID)
	}()

	chnInsertDoe := make(chan uint64)

	go func() {
		chnInsertDoe <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
			return db.Exec("INSERT INTO users (name) VALUES (?)", "Doe Smith")
		})
	}()
	insertDoeID := <-chnInsertDoe

	chnInsertMain := comfyMe.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec("INSERT INTO users (name) VALUES (?)", "John Doe")
	})
	<-comfyMe.WaitForChn(chnInsertMain)
	<-comfyMe.WaitForChn(insertDoeID)

	chnSelect := make(chan uint64)
	go func() {
		chnSelect <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
			names := []string{}
			rows, err := db.Query("SELECT name FROM users")
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var name string
			for rows.Next() {
				err := rows.Scan(&name)
				if err != nil {
					return nil, err
				}
				names = append(names, name)
			}
			return names, nil
		})
	}()

	selectMainID := comfyMe.New(func(db *sql.DB) (interface{}, error) {
		names := []string{}
		rows, err := db.Query("SELECT name FROM users")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var name string
		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, name)
		}
		return names, nil
	})
	resultMainUsers := <-comfyMe.WaitForChn(selectMainID)
	selectGoID := <-chnSelect // almost same time, see if we got our select from the previous goroutine
	resultFromGo := <-comfyMe.WaitForChn(selectGoID)
	var names []string

	switch dd := resultMainUsers.(type) {
	case error:
		t.Fatal(resultMainUsers)
	case []string:
		names = dd
		fmt.Println(names)
	default:
		t.Fatal("unexpected result")
	}
	slog.Info("Data read")
	var goNames []string

	switch dd := resultFromGo.(type) {
	case error:
		t.Fatal(resultFromGo)
	case []string:
		goNames = dd
		fmt.Println(goNames)
	default:
		t.Fatal("unexpected result")
	}
	slog.Info("Data read")

	// Compare names and goNames
	if len(names) != len(goNames) {
		t.Fatal("Data mismatch")
	}

	for i := 0; i < len(names); i++ {
		if names[i] != goNames[i] {
			t.Fatal("Data mismatch")
		}
	}

	if err := deleteTestDbFile(); err != nil {
		t.Fatal(err)
	}
}

const (
	setupSql = `
CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, user_name TEXT);
CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY, product_name TEXT);
DELETE FROM products;
`
	routines = 5000
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func randomSleep() {
	time.Sleep(time.Duration(r.Intn(5)) * time.Millisecond)
}

// trying to fix https://gist.github.com/mrnugget/0eda3b2b53a70fa4a894
// I know they are doing concurrent writes but that's the point of this test
// They want concurrent writes when I was the have the illusion of it
func TestLockedGist(t *testing.T) {

	comfyMe, err := New(
		WithMemory(),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer comfyMe.Close()

	done := make(chan struct{})

	id := comfyMe.New(func(db *sql.DB) (interface{}, error) {
		_, err := db.Exec(setupSql)
		return nil, err
	})
	<-comfyMe.WaitForChn(id)

	id = comfyMe.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec(`INSERT INTO products (product_name) VALUES ("computer")`)
	})
	<-comfyMe.WaitForChn(id)
	// comfyMe.Clear(id)

	writesIDs := []uint64{}
	readsIDs := []uint64{}

	insertWithID := func(id int) func(db *sql.DB) (interface{}, error) {
		return func(db *sql.DB) (interface{}, error) {
			fmt.Printf("+")
			return db.Exec(`INSERT INTO products (product_name) VALUES ( ? )`, fmt.Sprintf("product %d", id))
		}
	}

	go func() {
		// writes to users table
		for i := 0; i < routines; i++ {
			writesIDs = append(writesIDs, comfyMe.New(insertWithID(i)))
			randomSleep()
		}
		done <- struct{}{}
	}()

	go func() {
		// reads from products table, each read in separate go routine
		for i := 0; i < routines; i++ {
			go func(i, routines int) {
				readsIDs = append(readsIDs, comfyMe.New(func(db *sql.DB) (interface{}, error) {
					rows, err := db.Query("SELECT * FROM products WHERE id = 5")
					if err != nil {
						return nil, err
					}
					defer rows.Close()
					cols, _ := rows.Columns()

					values := []map[string]interface{}{}

					for rows.Next() {
						// Create a slice of interface{}'s to represent each column,
						// and a second slice to contain pointers to each item in the columns slice.
						columns := make([]interface{}, len(cols))
						columnPointers := make([]interface{}, len(cols))
						for i, _ := range columns {
							columnPointers[i] = &columns[i]
						}

						// Scan the result into the column pointers...
						if err := rows.Scan(columnPointers...); err != nil {
							return nil, err
						}

						// Create our map, and retrieve the value for each column from the pointers slice,
						// storing it in the map with the name of the column as the key.
						m := make(map[string]interface{})
						for i, colName := range cols {
							val := columnPointers[i].(*interface{})
							m[colName] = *val
						}

						// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
						values = append(values, m)
					}
					fmt.Printf(".")

					return values, nil
				}))
				done <- struct{}{}
			}(i, routines)

			randomSleep()
		}
	}()

	for i := 0; i < routines+1; i++ {
		<-done
	}

	// for _, v := range readsIDs {
	// 	result := <-comfyMe.WaitForChn(v)
	// 	switch dd := result.(type) {
	// 	case []map[string]interface{}:
	// 		// fmt.Println(dd)
	// 	}
	// }

}

func TestDatabaseLocking(t *testing.T) {
	t.Run("Test Lock Detection", func(t *testing.T) {
		// Create a new database
		db, err := New(WithMemory())
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// First check - database should be unlocked initially
		locked, err := db.IsLocked()
		if err != nil {
			t.Fatal(err)
		}
		if locked {
			t.Error("Database should not be locked initially")
		}

		// Create a table for testing
		createID := db.New(func(d *sql.DB) (interface{}, error) {
			_, err := d.Exec(`CREATE TABLE test_lock (
                id INTEGER PRIMARY KEY,
                value TEXT
            )`)
			return nil, err
		})
		if result := <-db.WaitForChn(createID); result != nil {
			if err, ok := result.(error); ok {
				t.Fatal(err)
			}
		}

		// Test concurrent access causing locks
		var wg sync.WaitGroup
		lockChan := make(chan bool, 1)

		// Start a long-running transaction in a goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()

			txID := db.New(func(d *sql.DB) (interface{}, error) {
				// Begin transaction
				tx, err := d.Begin()
				if err != nil {
					return nil, err
				}
				defer tx.Rollback()

				// Insert some data
				_, err = tx.Exec("INSERT INTO test_lock (value) VALUES (?)", "test-value")
				if err != nil {
					return nil, err
				}

				// Signal that we're in the middle of the transaction
				lockChan <- true

				// Hold the transaction open for a while
				time.Sleep(500 * time.Millisecond)

				// Commit the transaction
				return nil, tx.Commit()
			})
			<-db.WaitForChn(txID)
		}()

		// Wait for the transaction to start
		<-lockChan

		// Now check if database is locked
		locked, err = db.IsLocked()
		if err != nil {
			t.Fatal(err)
		}
		if !locked {
			t.Error("Database should be locked during transaction")
		}

		// Wait for the goroutine to finish
		wg.Wait()

		// Database should be unlocked again
		locked, err = db.IsLocked()
		if err != nil {
			t.Fatal(err)
		}
		if locked {
			t.Error("Database should be unlocked after transaction completes")
		}
	})

	t.Run("Test WaitUnlocked", func(t *testing.T) {
		db, err := New(WithMemory())
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// Start a long-running transaction
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()

			txID := db.New(func(d *sql.DB) (interface{}, error) {
				tx, err := d.Begin()
				if err != nil {
					return nil, err
				}
				defer tx.Rollback()

				// Do some work that will hold the lock
				_, err = tx.Exec(`
                    CREATE TABLE IF NOT EXISTS test_wait (id INTEGER PRIMARY KEY);
                    INSERT INTO test_wait DEFAULT VALUES;
                `)
				if err != nil {
					return nil, err
				}

				// Hold the lock for a while
				time.Sleep(time.Second)

				return nil, tx.Commit()
			})
			<-db.WaitForChn(txID)
		}()

		// Give the transaction time to start
		time.Sleep(100 * time.Millisecond)

		// Test WaitUnlocked with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		start := time.Now()
		err = db.WaitUnlocked(ctx)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatal(err)
		}
		if elapsed < time.Second/2 {
			t.Error("WaitUnlocked returned too quickly")
		}

		wg.Wait()
	})

	t.Run("Test WaitUnlocked Timeout", func(t *testing.T) {
		db, err := New(WithMemory())
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// Start a long-running transaction
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()

			txID := db.New(func(d *sql.DB) (interface{}, error) {
				tx, err := d.Begin()
				if err != nil {
					return nil, err
				}
				defer tx.Rollback()

				// Hold the lock for longer than the timeout
				_, err = tx.Exec(`
                    CREATE TABLE IF NOT EXISTS test_timeout (id INTEGER PRIMARY KEY);
                    INSERT INTO test_timeout DEFAULT VALUES;
                `)
				if err != nil {
					return nil, err
				}

				time.Sleep(2 * time.Second)

				return nil, tx.Commit()
			})
			<-db.WaitForChn(txID)
		}()

		// Give the transaction time to start
		time.Sleep(100 * time.Millisecond)

		// Test WaitUnlocked with a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err = db.WaitUnlocked(ctx)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
		if err != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded error, got: %v", err)
		}

		wg.Wait()
	})
}
