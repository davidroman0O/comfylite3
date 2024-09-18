# Comfylite3

Aren't you tired to be forced to write a different code for `sqlite3` because you can't use it with multiple goroutines? I was. I disliked the constraints and changing my habits.

That's why `comfylite3` exists! Just throw your queries at it and your `sql` will be executed*!

*: eventually!

![Gopher Comfy](./docs/gophercomfy.webp)


# Install 

```
go get -u github.com/davidroman0O/comfylite3
```

# sql.DB

`ComfyDB` is using all the functions of `sql.DB` so you can use as drop-in replacement!

# API

## Memory or File or What you want!

```go

// You want a default memory database
comfylite3.WithMemory()

// You want a default file database
comfylite3.File("comfyName.db")

// Feeling adventurous? You can!
comfylite3.WithConnection("file:/tmp/adventurousComfy.db?cache=shared")

```

## What you can do

Very simplistic API, `comfylite3` manage when to execute and you do as usual. I'm just judging how you will wrap that library!

```go

// Create a new comfy database for `sqlite3`
comfyDB, _ := comfylite3.New(comfylite3.WithMemory())

// Create future workload to cook
id := comfyDB.New(func(db *sql.DB) (interface{}, error) {
    _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
    return nil, err
})

// Ask it if it's done or not
if !comfyDB.IsDone(id) {
    fmt.Println("No no no! It is not done yet!")
}

// You will get what you send back! Error or not!
var result interface{}

// Eventually get your result
result = <-superComfy.WaitFor(id)

switch result.(type) {
	case error:
		fmt.Println("Oooh your query failed!", err)
}

```

## Using ComfyDB as a standard sql.DB

ComfyLite3 now provides an `OpenDB` function that allows you to use ComfyDB as a standard `sql.DB` instance. This makes it easier to integrate ComfyLite3 with existing code or libraries that expect a `*sql.DB`.

```go
// Create a new ComfyDB instance
comfy, err := comfylite3.New(comfylite3.WithMemory())
if err != nil {
    panic(err)
}

// Open a standard sql.DB instance using ComfyDB
db := comfylite3.OpenDB(comfy)

// Now you can use db as a regular *sql.DB
rows, err := db.Query("SELECT * FROM users")
// ...

// Don't forget to close both when you're done
defer db.Close()
defer comfy.Close()
```

The `OpenDB` function accepts additional options as variadic string arguments, allowing you to customize the connection string. For example:

```go
db := comfylite3.OpenDB(comfy, "_foreign_keys=on", "cache=shared")
```

This feature makes ComfyLite3 more flexible and easier to use in a variety of scenarios, especially when working with existing codebases or third-party libraries.

It can comes handy to integrate with other third-party like [ent](https://github.com/ent/ent), a powerful entity framework for Go. Here's how you can use ComfyLite3 as the underlying database for your ent client:

```go
import (
    "context"
    "log"

    "github.com/davidroman0O/comfylite3"
    "entgo.io/ent"
    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/sql"
)

func main() {
    // Create a new ComfyDB instance
    comfy, err := comfylite3.New(
        comfylite3.WithPath("./ent.db"),
    )
    if err != nil {
        log.Fatalf("failed creating ComfyDB: %v", err)
    }
    defer comfy.Close()

    // Use the OpenDB function to create a sql.DB instance with SQLite options
    db := comfylite3.OpenDB(
		comfy, 
		comfylite3.WithOption("_fk=1"),
		comfylite3.WithOption("cache=shared"),
		comfylite3.WithOption("mode=rwc"),
		comfylite3.WithForeignKeys(),
	)

    // Create a new ent client
    client := ent.NewClient(ent.Driver(sql.OpenDB(dialect.SQLite, db)))
    defer client.Close()

    ctx := context.Background()

    // Run the auto migration tool
    if err := client.Schema.Create(ctx); err != nil {
        log.Fatalf("failed creating schema resources: %v", err)
    }

    // Your ent operations go here
    // For example:
    // user, err := client.User.Create().SetName("John Doe").Save(ctx)
    // if err != nil {
    //     log.Fatalf("failed creating user: %v", err)
    // }
    // fmt.Printf("User created: %v\n", user)
}
```

In this setup:

1. We create a ComfyDB instance with a file-based SQLite database.
2. We use `OpenDB` to create a standard `*sql.DB` instance, passing SQLite-specific options.
3. We create an ent client using the SQLite dialect and our ComfyDB-backed `*sql.DB`.
4. We run ent's auto-migration to create the schema.

This integration allows you to leverage the concurrency benefits of ComfyLite3 while using ent's powerful ORM features. [Check by yourself that repository](https://github.com/davidroman0O/comfylite3-ent)

## Migrations

Migrations is important and `sqlite` is a specific type of database, and it support migrations!

```go

// Let's imagine a set of migrations
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

// create and add your migrations
comfyDB, _ := comfylite3.New(
		comfylite3.WithMemory(),
		comfylite3.WithMigration(memoryMigrations...),
		comfylite3.WithMigrationTableName("_migrations"), // even customize your migration table!
	)

// Migrations Up and Down are easy

// Up to the top!
if err := comfyDB.Up(context.Background()); err != nil {
	panic(err)
}

// Specify how many down you want to do
if err := comfyDB.Down(context.Background(), 1); err != nil {
	panic(err)
}

comfyDB.Version() // return all the existing versions []uint
comfyDB.Index() // return the current index of the migration
comfyDB.ShowTables() // return all table names
comfyDB.ShowColumns("name") // return columns data of one table

```


# Example 

```go
package main 

import (
    "fmt"
    "github.com/davidroman0O/comfylite3"
)

func main() {

	var superComfy *comfylite3.ComfyDB
	var err error

    // Make yourself comfy
	if superComfy, err = comfylite3.New(comfylite3.WithMemory()); err != nil {
		panic(err)
	}

	defer superComfy.Close()

    // Ask it to make a query for you
    ticket := superComfy.New(func(db *sql.DB) (interface{}, error) {
		_, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
		return nil, err
	})

    // It will send you the result when it's finished cooking
	<-superComfy.WaitFor(ticket)

    // You don't have to change your habit, just use goroutines
    // `comfylite` will cook for you
    var futureTicket uint64
    go func() {
        futureTicket = superComfy.New(func(db *sql.DB) (interface{}, error) {
            _, err := db.Exec("INSERT INTO users (name) VALUES (?)", fmt.Sprintf("user%d", 1))
            return nil, err
        })
    }()

    if !superComfy.IsDone(futureTicket) {
        fmt.Println("No no no! It is not done yet!")
    }

    // Let comfylite3 cook!
    <-superComfy.WaitFor(futureTicket)
}

```

Or even with a more complex example! Here we want to compute the average amount of inserts per second:

```go
package main 

import (
    "fmt"
    "github.com/davidroman0O/comfylite3"
)

func main() {
   
	var superComfy *comfylite3.ComfyDB
	var err error
	if superComfy, err = comfylite3.New(comfylite3.WithMemory()); err != nil {
		panic(err)
	}

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

```

Enjoy freeing yourself from `database is locked`!
