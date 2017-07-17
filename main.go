package main

import (
	"log"

	"github.com/asdine/storm"
	"github.com/namsral/flag"
)

var (
	db *storm.DB
)

func main() {
	var (
		dbpath string
		bind   string
	)

	flag.StringVar(&dbpath, "dbpath", "todo.db", "Database path")
	flag.StringVar(&bind, "bind", "0.0.0.0:8000", "[int]:<port> to bind to")
	flag.Parse()

	var err error
	db, err = storm.Open(dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	NewServer(bind).ListenAndServe()
}
