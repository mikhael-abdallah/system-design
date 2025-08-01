// app/benchmark.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// runBenchmarks orchestrates the different performance tests.
func runBenchmarks(db *sql.DB, bf *BloomFilter) {
	log.Println("\n--- Preparing data for benchmarks ---")

	// Prepare a slice of 100,000 existing IDs
	existingIDs := make([][]byte, 0, benchmark_n)
	rows, err := db.Query("SELECT id FROM users LIMIT $1", benchmark_n)
	if err != nil {
		log.Fatalf("Failed to fetch existing IDs for benchmark: %v", err)
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			log.Printf("Error scanning existing ID: %v", err)
			continue
		}
		existingIDs = append(existingIDs, id[:])
	}
	rows.Close()
	log.Printf("Fetched %d existing IDs for testing.", len(existingIDs))

	// Prepare a slice of 100,000 non-existent IDs
	nonExistentIDs := make([][]byte, 0, benchmark_n)
	for i := 0; i < benchmark_n; i++ {
		id := uuid.New()
		nonExistentIDs = append(nonExistentIDs, id[:])
	}
	log.Printf("Generated %d non-existent IDs for testing.", len(nonExistentIDs))

	// Run the benchmarks
	benchmarkNonExistentUsers(db, bf, nonExistentIDs)
	benchmarkExistingUsers(db, bf, existingIDs)
}

// benchmarkNonExistentUsers compares lookup times for items that are not in the set.
func benchmarkNonExistentUsers(db *sql.DB, bf *BloomFilter, idsToTest [][]byte) {
	fmt.Println("\n-------------------------------------------------------------")
	log.Printf("--- Benchmark: Non-Existent Users (%d lookups) ---", len(idsToTest))
	fmt.Println("-------------------------------------------------------------")

	// --- Test 1: Using the Bloom Filter ---
	falsePositives := 0
	start := time.Now()
	for _, id := range idsToTest {
		if bf.Test(id) {
			// This is a potential false positive. In a real app, we would now query the DB.
			// For this benchmark, we just count it to measure the filter's accuracy.
			falsePositives++
		}
	}
	durationWithFilter := time.Since(start)
	
	fmt.Println("[With Bloom Filter]")
	printMetrics(durationWithFilter, len(idsToTest))
	fpRate := (float64(falsePositives) / float64(len(idsToTest))) * 100
	fmt.Printf("  False Positives:  %d (%.4f%%)\n", falsePositives, fpRate)


	// --- Test 2: Querying the Database Directly ---
	start = time.Now()
	for _, idBytes := range idsToTest {
		var id uuid.UUID
		copy(id[:], idBytes)
		// We expect this to always return sql.ErrNoRows
		db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
	}
	durationWithDB := time.Since(start)

	fmt.Println("\n[Database Only]")
	printMetrics(durationWithDB, len(idsToTest))

	fmt.Printf("\nConclusion: Bloom Filter was %.2fx faster for non-existent items.\n", float64(durationWithDB)/float64(durationWithFilter))
}

// benchmarkExistingUsers compares lookup times for items that are in the set.
func benchmarkExistingUsers(db *sql.DB, bf *BloomFilter, idsToTest [][]byte) {
	fmt.Println("\n-------------------------------------------------------------")
	log.Printf("--- Benchmark: Existing Users (%d lookups) ---", len(idsToTest))
	fmt.Println("-------------------------------------------------------------")
	
	// --- Test 1: Using Bloom Filter + Database Query ---
	start := time.Now()
	for _, idBytes := range idsToTest {
		// Step 1: Check the filter (this is the overhead)
		if bf.Test(idBytes) {
			// Step 2: Query the DB (this will always happen for existing items)
			var id uuid.UUID
			copy(id[:], idBytes)
			db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
		}
	}
	durationWithFilter := time.Since(start)
	
	fmt.Println("[Bloom Filter + Database]")
	printMetrics(durationWithFilter, len(idsToTest))

	// --- Test 2: Querying the Database Directly ---
	start = time.Now()
	for _, idBytes := range idsToTest {
		var id uuid.UUID
		copy(id[:], idBytes)
		db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
	}
	durationWithDB := time.Since(start)

	fmt.Println("\n[Database Only]")
	printMetrics(durationWithDB, len(idsToTest))

	overhead := durationWithFilter - durationWithDB
	fmt.Printf("\nConclusion: The Bloom Filter added an average overhead of %v per lookup for existing items.\n", overhead/time.Duration(len(idsToTest)))
}

// printMetrics is a helper function to display performance results.
func printMetrics(duration time.Duration, numOps int) {
	avg := duration / time.Duration(numOps)
	opsPerSec := float64(numOps) / duration.Seconds()
	
	fmt.Printf("  Total Time:       %v\n", duration)
	fmt.Printf("  Avg. Per Lookup:  %v\n", avg)
	fmt.Printf("  Ops/Second:       %.2f\n", opsPerSec)
}