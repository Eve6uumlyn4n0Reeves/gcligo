package main

import (
	"database/sql"
	"flag"
	"fmt"
	stdlog "log"
	"os"

	"gcli2api-go/internal/migrations"

	_ "github.com/lib/pq"
)

func main() {
	dsn := flag.String("dsn", "", "PostgreSQL connection string")
	action := flag.String("action", "up", "migration action: up, down, or version")
	steps := flag.Int("steps", 1, "steps to migrate when action=down")
	flag.Parse()

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: -dsn")
		os.Exit(2)
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		stdlog.Fatalf("open database: %v", err)
	}
	defer db.Close()

	switch *action {
	case "up":
		if err := migrations.PostgresUp(db); err != nil {
			stdlog.Fatalf("migrate up: %v", err)
		}
		stdlog.Println("migrations applied")
	case "down":
		if err := migrations.PostgresDown(db, *steps); err != nil {
			stdlog.Fatalf("migrate down: %v", err)
		}
		stdlog.Printf("rolled back %d step(s)\n", *steps)
	case "version":
		version, dirty, err := migrations.PostgresVersion(db)
		if err != nil {
			stdlog.Fatalf("read version: %v", err)
		}
		state := "clean"
		if dirty {
			state = "dirty"
		}
		stdlog.Printf("current version: %d (%s)\n", version, state)
	default:
		fmt.Fprintf(os.Stderr, "unknown action %q (expected up, down, version)\n", *action)
		os.Exit(2)
	}
}
