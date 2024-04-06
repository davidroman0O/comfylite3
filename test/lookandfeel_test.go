package test

import (
	"database/sql"
	"testing"

	"github.com/davidroman0O/comfylite3"
)

func TestMemory(t *testing.T) {
	comfylite3.Initialize(
		comfylite3.WithMemory(),
	)
	defer comfylite3.Close()

	id := comfylite3.New(func(db *sql.DB) (interface{}, error) {
		return db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
	})

	<-comfylite3.WaitFor(id)
}
