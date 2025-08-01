package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// User represents our table in the database
type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// A larger data field to help fill the DB's RAM
	ProfileData string `json:"profile_data"` 
}

// connectDB establishes a connection to the PostgreSQL database
func connectDB() *sql.DB {
	connStr := "postgres://user:password@db:5432/mydb?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	return db
}

// seedDatabase inserts a large number of users for performance testing
func seedDatabase(db *sql.DB, n int) {
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	log.Printf("The database already contains %d users.", userCount)
	if userCount >= n / 2 {
		log.Printf("The database already contains %d users. Skipping insertion.", userCount)
		return
	}

	log.Printf("Starting the insertion of %d users into the database. This may take a long time...", n)
	startTime := time.Now()

	conn, err := pgx.Connect(context.Background(), "postgres://user:password@db:5432/mydb")
	if err != nil {
		log.Fatalf("Error getting pgx connection: %v", err)
	}
	defer conn.Close(context.Background())

	copySource := pgx.CopyFromFunc(func() ([]any, error) {
		if userCount >= n {
			return nil, io.EOF 
		}
		userCount++
		if userCount % 1000 == 0 {
			log.Printf("Inserting user %d", userCount)
		}
		id := uuid.New()
		name := fmt.Sprintf("User %d", userCount)
		profileData := fmt.Sprintf("Profile data for %s. ", name)
		
		return []any{id, name, profileData}, nil
	})

	_, err = conn.CopyFrom(context.Background(), pgx.Identifier{"users"}, []string{"id", "name", "profile_data"}, copySource)
	if err != nil {
		log.Fatalf("Error during bulk copy operation: %v", err)
	}

	log.Printf("Insertion of %d users completed in %v", n, time.Since(startTime))
}
