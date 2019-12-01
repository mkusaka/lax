package db

import (
	"log"
	"testing"
	"time"
)

// database launcher
// database cleaner

func databaseCleaner() {
	c := NewClient(1000 * time.Millisecond)
	err := c.database.Drop(c.defaultContext)
	if err != nil {
		log.Fatal(err)
	}
}

// database seed
func TestTODO(t *testing.T) {
}
