package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	cuckoo "github.com/seiflotfy/cuckoofilter"
)

const (
	// Using 20 million items as the target dataset size
	n_items = 20_000_000
	
	// Bloom Filter parameters for n=20M, p=1%
	m_bits   = 191_701_179 // ~23 MB
	k_hashes = 7

	// Cuckoo Filter capacity (next power of 2 >= n_items)
	// 2^25 = 33,554,432
	cuckoo_capacity = 67_108_864

	benchmark_n = 100_000 // Number of lookups for each benchmark
)

func main() {
	// 1. Connect to DB and seed if necessary (code remains the same)
	db := connectDB()
	defer db.Close()
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY, name TEXT, profile_data TEXT)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	seedDatabase(db, n_items)

	// 3. Create both filters
	log.Println("Creating Bloom and Cuckoo filters in memory...")
	bloomFilter := NewBloomFilter(m_bits, k_hashes)
	cuckooFilter := cuckoo.NewFilter(cuckoo_capacity)

	log.Println("Warming up both filters with data from the DB. This may take a while...")
	startTime := time.Now()

	rows, err := db.Query("SELECT id FROM users")
	if err != nil {
		log.Fatalf("Failed to fetch IDs for filter warm-up: %v", err)
	}
	defer rows.Close()

	var id uuid.UUID
	count := 0
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			log.Printf("Error scanning ID: %v", err)
			continue
		}
		// Add the same ID to both filters
		idBytes := id[:]
		bloomFilter.Add(idBytes)
		cuckooFilter.Insert(idBytes)
		count++

		if count%5_000_000 == 0 {
			log.Printf("... %d million IDs added to filters", count/1_000_000)
		}
	}
	log.Printf("Filters warmed up with %d items in %v.", count, time.Since(startTime))

	// 4. Run the comparative benchmarks
	runBenchmarks(db, bloomFilter, cuckooFilter)
}