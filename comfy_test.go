package comfylite3

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMemory(t *testing.T) {

	comfyMe, err := Comfy(
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
	<-comfyMe.WaitFor(createID)

	go func() {
		chnInsert := make(chan uint64)
		go func() {
			chnInsert <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
				return db.Exec("INSERT INTO users (name) VALUES (?)", "Jane Smith")
			})
		}()
		insertID := <-chnInsert
		<-comfyMe.WaitFor(insertID)
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
	<-comfyMe.WaitFor(chnInsertMain)
	<-comfyMe.WaitFor(insertDoeID)

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
	resultMainUsers := <-comfyMe.WaitFor(selectMainID)
	selectGoID := <-chnSelect // almost same time, see if we got our select from the previous goroutine
	resultFromGo := <-comfyMe.WaitFor(selectGoID)
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

	comfyMe, err := Comfy(
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
	<-comfyMe.WaitFor(createID)

	go func() {
		chnInsert := make(chan uint64)
		go func() {
			chnInsert <- comfyMe.New(func(db *sql.DB) (interface{}, error) {
				return db.Exec("INSERT INTO users (name) VALUES (?)", "Jane Smith")
			})
		}()
		insertID := <-chnInsert
		<-comfyMe.WaitFor(insertID)
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
	<-comfyMe.WaitFor(chnInsertMain)
	<-comfyMe.WaitFor(insertDoeID)

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
	resultMainUsers := <-comfyMe.WaitFor(selectMainID)
	selectGoID := <-chnSelect // almost same time, see if we got our select from the previous goroutine
	resultFromGo := <-comfyMe.WaitFor(selectGoID)
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

	comfyMe, err := Comfy(
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
	<-comfyMe.WaitFor(id)

	id = comfyMe.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec(`INSERT INTO products (product_name) VALUES ("computer")`)
	})
	<-comfyMe.WaitFor(id)
	comfyMe.Clear(id)

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
	// 	result := <-comfyMe.WaitFor(v)
	// 	switch dd := result.(type) {
	// 	case []map[string]interface{}:
	// 		// fmt.Println(dd)
	// 	}
	// }

}
