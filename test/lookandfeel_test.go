package test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/davidroman0O/comfylite3"
)

// All migrations of the memory database
var memoryMigrations []comfylite3.Migration = []comfylite3.Migration{
	comfylite3.NewMigration(
		1,
		"genesis",
		func(tx *sql.Tx) error {
			if _, err := tx.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
				return err
			}
			if _, err := tx.Exec("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, user_id INTEGER, FOREIGN KEY(user_id) REFERENCES users(id))"); err != nil {
				return err
			}
			return nil
		},
		func(tx *sql.Tx) error {
			// undo previous up function
			if _, err := tx.Exec("DROP TABLE users"); err != nil {
				return err
			}
			if _, err := tx.Exec("DROP TABLE products"); err != nil {
				return err
			}
			return nil
		}),
	comfylite3.NewMigration(
		2,
		"new_table",
		func(tx *sql.Tx) error {
			if _, err := tx.Exec("ALTER TABLE products ADD COLUMN new_column TEXT"); err != nil {
				return err
			}
			if _, err := tx.Exec("CREATE TABLE new_table (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
				return err
			}
			return nil
		},
		func(tx *sql.Tx) error {
			// undo previous up function
			if _, err := tx.Exec("ALTER TABLE products DROP COLUMN new_column"); err != nil {
				return err
			}
			if _, err := tx.Exec("DROP TABLE new_table"); err != nil {
				return err
			}
			return nil
		}),
	comfylite3.NewMigration(
		3,
		"new_random",
		func(tx *sql.Tx) error {
			// remove new_column from products
			if _, err := tx.Exec("ALTER TABLE products DROP COLUMN new_column"); err != nil {
				return err
			}
			return nil
		},
		func(tx *sql.Tx) error {
			// add new_column to products
			if _, err := tx.Exec("ALTER TABLE products ADD COLUMN new_column TEXT"); err != nil {
				return err
			}
			return nil
		},
	),
}

func TestMigration(t *testing.T) {

	var superComfy *comfylite3.ComfyDB
	var err error
	if superComfy, err = comfylite3.New(
		// comfylite3.WithPath("./test.db"),
		comfylite3.WithMemory(),
		comfylite3.WithMigration(memoryMigrations...),
	); err != nil {
		t.Fatal(err)
	}

	defer superComfy.Close()

	var version uint
	if version, err = superComfy.Version(); err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("expected version 0, got %d", version)
	}

	var index []uint
	if index, err = superComfy.Index(); err != nil {
		t.Fatal(err)
	}

	if len(index) != 0 {
		t.Fatalf("expected index 0, got %d", len(index))
	}

	var tables []string

	if tables, err = superComfy.ShowTables(); err != nil {
		t.Fatal(err)
	}

	if len(tables) == 0 {
		t.Fatalf("expected tables, got %d", len(tables))
	}

	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}

	for _, table := range tables {
		if table != "_migrations" && table != "sqlite_sequence" {
			t.Fatalf("expected _migrations or sqlite_sequence, got %s", table)
		}
	}

	// Attempt to go up to the top
	if err = superComfy.Up(context.Background()); err != nil {
		t.Fatal(err)
	}

	if tables, err = superComfy.ShowTables(); err != nil {
		t.Fatal(err)
	}

	if len(tables) == 0 {
		t.Fatalf("expected tables, got %d", len(tables))
	}

	if len(tables) != 5 {
		fmt.Println(tables)
		t.Fatalf("expected 5 tables, got %d", len(tables))
	}

	for _, table := range tables {
		if table != "_migrations" && table != "sqlite_sequence" && table != "users" && table != "products" && table != "new_table" {
			t.Fatalf("expected _migrations or sqlite_sequence, got %s", table)
		}
		switch table {
		case "users":
			var userCols []comfylite3.Column
			if userCols, err = superComfy.ShowColumns("users"); err != nil {
				t.Fatal(err)
			}
			fmt.Println(userCols)
			if len(userCols) == 0 {
				t.Fatalf("expected columns, got %d", len(userCols))
			}
			for _, col := range userCols {
				switch col.Name {
				case "id":
					if col.Type != "INTEGER" {
						t.Fatalf("expected INTEGER, got %s", col.Type)
					}
					if col.Pk != true {
						t.Fatalf("expected PRIMARY KEY, got %v", col.Pk)
					}
				case "name":
					if col.Type != "TEXT" {
						t.Fatalf("expected TEXT, got %s", col.Type)
					}
				default:
					t.Fatalf("unexpected column %s", col.Name)
				}
			}
		case "products":
			var productCols []comfylite3.Column
			if productCols, err = superComfy.ShowColumns("products"); err != nil {
				t.Fatal(err)
			}
			fmt.Println(productCols)
			if len(productCols) == 0 {
				t.Fatalf("expected columns, got %d", len(productCols))
			}
			for _, col := range productCols {
				switch col.Name {
				case "id":
					if col.Type != "INTEGER" {
						t.Fatalf("expected INTEGER, got %s", col.Type)
					}
					if col.Pk != true {
						t.Fatalf("expected PRIMARY KEY, got %v", col.Pk)
					}
				case "name":
					if col.Type != "TEXT" {
						t.Fatalf("expected TEXT, got %s", col.Type)
					}
				case "user_id":
					if col.Type != "INTEGER" {
						t.Fatalf("expected INTEGER, got %s", col.Type)
					}
					// if col.ForeignKey != "users(id)" {
					// 	t.Fatalf("expected FOREIGN KEY users(id), got %s", col.ForeignKey)
					// }
				case "new_column":
					if col.Type != "TEXT" {
						t.Fatalf("expected TEXT, got %s", col.Type)
					}
				default:
					t.Fatalf("unexpected column %s", col.Name)
				}
			}
		case "new_table":
			var newTableCols []comfylite3.Column
			if newTableCols, err = superComfy.ShowColumns("new_table"); err != nil {
				t.Fatal(err)
			}
			fmt.Println(newTableCols)
			if len(newTableCols) == 0 {
				t.Fatalf("expected columns, got %d", len(newTableCols))
			}
			for _, col := range newTableCols {
				switch col.Name {
				case "id":
					if col.Type != "INTEGER" {
						t.Fatalf("expected INTEGER, got %s", col.Type)
					}
					if col.Pk != true {
						t.Fatalf("expected PRIMARY KEY, got %v", col.Pk)
					}
				case "name":
					if col.Type != "TEXT" {
						t.Fatalf("expected TEXT, got %s", col.Type)
					}
				default:
					t.Fatalf("unexpected column %s", col.Name)
				}
			}
		case "_migrations":
			var migrationsCols []comfylite3.Column
			if migrationsCols, err = superComfy.ShowColumns("_migrations"); err != nil {
				t.Fatal(err)
			}
			fmt.Println(migrationsCols)
			if len(migrationsCols) == 0 {
				t.Fatalf("expected columns, got %d", len(migrationsCols))
			}
			for _, col := range migrationsCols {
				switch col.Name {
				case "id":
					if col.Type != "INTEGER" {
						t.Fatalf("expected INTEGER, got %s", col.Type)
					}
					if col.Pk != true {
						t.Fatalf("expected PRIMARY KEY, got %v", col.Pk)
					}
				case "description":
					if col.Type != "VARCHAR(255)" {
						t.Fatalf("expected VARCHAR(255), got %s", col.Type)
					}
				case "version":
					if col.Type != "INTEGER" {
						t.Fatalf("expected INTEGER, got %s", col.Type)
					}
				default:
					t.Fatalf("unexpected column %s", col.Name)
				}
			}
		case "sqlite_sequence":
			// don't care
		default:
			t.Fatalf("unexpected table %s", table)
		}
	}

	if err = superComfy.Down(context.Background(), 2); err != nil {
		t.Fatal(err)
	}

	if err = superComfy.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestMemory(t *testing.T) {

	var superComfy *comfylite3.ComfyDB
	var err error
	if superComfy, err = comfylite3.New(
		comfylite3.WithMemory(),
	); err != nil {
		panic(err)
	}

	defer superComfy.Close()
	ticket := superComfy.New(func(db *sql.DB) (interface{}, error) {
		_, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
		return nil, err
	})
	<-superComfy.WaitForChn(ticket)

	done := make(chan struct{})

	go func() {
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		for range 10000 {
			ticket := superComfy.New(func(db *sql.DB) (interface{}, error) {
				_, err := db.Exec("INSERT INTO users (name) VALUES (?)", fmt.Sprintf("user%d", 1))
				return nil, err
			})
			<-superComfy.WaitForChn(ticket)
			// simulate random insert with a random sleep
			time.Sleep(time.Duration(random.Intn(5)) * time.Millisecond)
		}
		done <- struct{}{}
	}()

	// Let's measure how many records per second
	metrics := []int{}

	ticker := time.NewTicker(1 * time.Second)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	compute := true
	for compute {
		select {
		case <-done:
			ticker.Stop()
			cancel()
			compute = false
		case <-ctx.Done():
			ticker.Stop()
			cancel()
			compute = false
		case <-ticker.C:
			ticket := superComfy.New(func(db *sql.DB) (interface{}, error) {
				rows, err := db.Query("SELECT COUNT(*) FROM users")
				if err != nil {
					return nil, err
				}
				defer rows.Close()

				var count int
				if rows.Next() {
					err = rows.Scan(&count)
					if err != nil {
						return nil, err
					}
				}

				return count, err
			})
			result := <-superComfy.WaitForChn(ticket)
			metrics = append(metrics, result.(int))
		}
	}

	total := 0
	for _, value := range metrics {
		total += value
	}
	average := float64(total) / float64(len(metrics))
	fmt.Printf("Average: %.2f\n", average)
}

// func TestSqlDB(t *testing.T) {
// 	var db *sql.DB

// 	c, e := comfylite3.New(comfylite3.WithMemory())
// 	if e != nil {
// 		t.Fatal(e)
// 	}

// }
