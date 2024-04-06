# Comfylite3

Aren't you tired to be forced to write a different code for `sqlite3` because you can't use it with multiple goroutines? I was. I disliked the constraints and changing my habits.

That's why `comfylite3` exists! Just throw your queries at it and your `sql` will be executed*!

*: eventually!

![Gopher Comfy](./docs/gopher%20comfy.webp)


# Install 

```
go get -u github.com/davidroman0O/comfylite3
```

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

```go

// Create a new comfy database for `sqlite3`
comfyDB, _ := comfylite3.Comfy(comfylite3.WithMemory())

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
	if superComfy, err = comfylite3.Comfy(comfylite3.WithMemory()); err != nil {
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
	if superComfy, err = comfylite3.Comfy(comfylite3.WithMemory()); err != nil {
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
