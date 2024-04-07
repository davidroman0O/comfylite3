package test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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
	if superComfy, err = comfylite3.Comfy(
		comfylite3.WithMemory(),
		comfylite3.WithMigration(memoryMigrations...),
	); err != nil {
		panic(err)
	}

	defer superComfy.Close()

	slog.Info("get version")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Version())
	slog.Info("get index")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Index())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going up")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Up(context.Background()))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going up")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Up(context.Background()))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going down")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Down(context.Background(), 1))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going down")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Down(context.Background(), 1))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going up")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Up(context.Background()))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going down")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Down(context.Background(), 1))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
	slog.Info("going up")
	fmt.Println(superComfy.ShowTables())
	fmt.Println(superComfy.Up(context.Background()))
	fmt.Println(superComfy.Migrations())
	fmt.Println(superComfy.ShowTables())
}

func TestMemory(t *testing.T) {

	var superComfy *comfylite3.ComfyDB
	var err error
	if superComfy, err = comfylite3.Comfy(
		comfylite3.WithMemory(),
	); err != nil {
		panic(err)
	}

	defer superComfy.Close()
	ticket := superComfy.New(func(db *sql.DB) (interface{}, error) {
		_, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
		return nil, err
	})
	<-superComfy.WaitFor(ticket)

	done := make(chan struct{})

	go func() {
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		for range 10000 {
			ticket := superComfy.New(func(db *sql.DB) (interface{}, error) {
				_, err := db.Exec("INSERT INTO users (name) VALUES (?)", fmt.Sprintf("user%d", 1))
				return nil, err
			})
			<-superComfy.WaitFor(ticket)
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
			result := <-superComfy.WaitFor(ticket)
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
