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

`ComfyDB` is using all the functions of `sql.DB` so you can use as drop-in replacement! It now uses retrypool under the hood for better reliability and concurrent operation handling.

# API

## Memory or File or What you want!

```go
// You want a default memory database
comfylite3.WithMemory()

// You want a default file database
comfylite3.WithPath("comfyName.db")

// Feeling adventurous? You can!
comfylite3.WithConnection("file:/tmp/adventurousComfy.db?cache=shared")
```

## Retry Configuration

```go
comfy, err := comfylite3.New(
    comfylite3.WithMemory(),
    comfylite3.WithRetryAttempts(3),        // Configure max retries
    comfylite3.WithRetryDelay(time.Second), // Set delay between retries
    comfylite3.WithPanicHandler(func(v interface{}, stackTrace string) {
        // Custom panic handling
    }),
)
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
db := comfylite3.OpenDB(comfy, 
    comfylite3.WithOption("_fk=1"),
    comfylite3.WithForeignKeys(),
)

// Now you can use db as a regular *sql.DB
rows, err := db.Query("SELECT * FROM users")
// ...

// Don't forget to close both when you're done
defer db.Close()
defer comfy.Close()
```

This feature makes ComfyLite3 more flexible and easier to use in a variety of scenarios, especially when working with existing codebases or third-party libraries.

## What you can do

Very simplistic API, `comfylite3` manage when to execute and you do as usual.

```go
// Create a new comfy database for `sqlite3`
comfyDB, _ := comfylite3.New(comfylite3.WithMemory())

// Create future workload to cook
id := comfyDB.New(func(db *sql.DB) (interface{}, error) {
    _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
    return nil, err
})

// You will get what you send back! Error or not!
result := <-comfyDB.WaitForChn(id)

switch result.(type) {
    case error:
        fmt.Println("Oooh your query failed!", result)
}
```

## Integration with Ent

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

comfyDB.Version()  // return all the existing versions []uint
comfyDB.Index()    // return the current index of the migration
comfyDB.ShowTables() // return all table names
comfyDB.ShowColumns("name") // return columns data of one table
```

# Example with Metrics

Here's a more complex example that computes the average amount of inserts per second:

```go
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
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
```

Enjoy freeing yourself from `database is locked`!