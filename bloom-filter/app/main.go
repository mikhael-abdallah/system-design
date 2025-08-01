// app/main.go
package main

import (
	"log"
	"time"

	"github.com/google/uuid"
)

const (
	// Parameters calculated for n=50M and p=1%
	n_items      = 20_000_000
	m_bits       = 192_000_000
	k_hashes     = 7
	benchmark_n  = 100_000 // Number of lookups for each benchmark
)

func main() {
	// 1. Connect to the database and ensure the table exists
	db := connectDB()
	defer db.Close()
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		name TEXT,
		profile_data TEXT
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 2. Seed the database with a large volume of data (if necessary)
	seedDatabase(db, n_items)

	// 3. Create and "warm up" the Bloom Filter
	log.Println("Creating the Bloom Filter in memory...")
	bloomFilter := NewBloomFilter(m_bits, k_hashes)

	log.Println("Warming up the Bloom Filter with data from the DB. This may take a while...")
	startTime := time.Now()

	rows, err := db.Query("SELECT id FROM users")
	if err != nil {
		log.Fatalf("Failed to fetch IDs for Bloom Filter warm-up: %v", err)
	}
	defer rows.Close()

	var id uuid.UUID
	count := 0
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			log.Printf("Error scanning ID: %v", err)
			continue
		}
		bloomFilter.Add(id[:])
		count++
		if count%5_000_000 == 0 {
			log.Printf("... %d million IDs added to the filter", count/1_000_000)
		}
	}

	log.Printf("Bloom Filter warmed up with %d items in %v.", count, time.Since(startTime))

	// 4. Run the performance benchmarks
	runBenchmarks(db, bloomFilter)
}