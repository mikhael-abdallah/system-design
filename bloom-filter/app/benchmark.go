package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	cuckoo "github.com/seiflotfy/cuckoofilter"
)

// runBenchmarks orchestrates the different performance tests for both filters.
func runBenchmarks(db *sql.DB, bf *BloomFilter, cf *cuckoo.Filter) {
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
	benchmarkNonExistentUsers(db, bf, cf, nonExistentIDs)
	benchmarkExistingUsers(db, bf, cf, existingIDs)
	benchmarkDeletions(cf, existingIDs) // Deletion is only possible with Cuckoo Filter
}

// --- Benchmark for Non-Existent Items ---
func benchmarkNonExistentUsers(db *sql.DB, bf *BloomFilter, cf *cuckoo.Filter, idsToTest [][]byte) {
	fmt.Println("\n-------------------------------------------------------------")
	log.Printf("--- Benchmark: Non-Existent Users (%d lookups) ---", len(idsToTest))
	fmt.Println("-------------------------------------------------------------")

	// Test 1: Bloom Filter
	bfFalsePositives := 0
	startBf := time.Now()
	for _, id := range idsToTest {
		if bf.Test(id) {
			bfFalsePositives++
		}
	}
	durationBf := time.Since(startBf)
	fmt.Println("[Bloom Filter]")
	printMetrics(durationBf, len(idsToTest))
	fpRateBf := (float64(bfFalsePositives) / float64(len(idsToTest))) * 100
	fmt.Printf("  False Positives:  %d (%.4f%%)\n", bfFalsePositives, fpRateBf)

	// Test 2: Cuckoo Filter
	cfFalsePositives := 0
	startCf := time.Now()
	for _, id := range idsToTest {
		if cf.Lookup(id) {
			cfFalsePositives++
		}
	}
	durationCf := time.Since(startCf)
	fmt.Println("\n[Cuckoo Filter]")
	printMetrics(durationCf, len(idsToTest))
	fpRateCf := (float64(cfFalsePositives) / float64(len(idsToTest))) * 100
	fmt.Printf("  False Positives:  %d (%.4f%%)\n", cfFalsePositives, fpRateCf)


	// Test 3: Database Only
	startDb := time.Now()
	for _, idBytes := range idsToTest {
		var id uuid.UUID
		copy(id[:], idBytes)
		db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
	}
	durationDb := time.Since(startDb)
	fmt.Println("\n[Database Only]")
	printMetrics(durationDb, len(idsToTest))

	fmt.Printf("\nConclusion: Cuckoo was %.2fx faster than Bloom. Bloom was %.2fx faster than DB.\n", float64(durationBf)/float64(durationCf), float64(durationDb)/float64(durationBf))
}

// --- Benchmark for Existing Items ---
func benchmarkExistingUsers(db *sql.DB, bf *BloomFilter, cf *cuckoo.Filter, idsToTest [][]byte) {
	fmt.Println("\n-------------------------------------------------------------")
	log.Printf("--- Benchmark: Existing Users (%d lookups) ---", len(idsToTest))
	fmt.Println("-------------------------------------------------------------")
	
	// Test 1: Bloom Filter + DB
	startBf := time.Now()
	for _, idBytes := range idsToTest {
		if bf.Test(idBytes) {
			var id uuid.UUID; copy(id[:], idBytes); db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
		}
	}
	durationBf := time.Since(startBf)
	fmt.Println("[Bloom Filter + Database]")
	printMetrics(durationBf, len(idsToTest))

	// Test 2: Cuckoo Filter + DB
	startCf := time.Now()
	for _, idBytes := range idsToTest {
		if cf.Lookup(idBytes) {
			var id uuid.UUID; copy(id[:], idBytes); db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
		}
	}
	durationCf := time.Since(startCf)
	fmt.Println("\n[Cuckoo Filter + Database]")
	printMetrics(durationCf, len(idsToTest))

	// Test 3: Database Only
	startDb := time.Now()
	for _, idBytes := range idsToTest {
		var id uuid.UUID; copy(id[:], idBytes); db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&id)
	}
	durationDb := time.Since(startDb)
	fmt.Println("\n[Database Only]")
	printMetrics(durationDb, len(idsToTest))
	
	overheadBf := durationBf - durationDb
	overheadCf := durationCf - durationDb
	fmt.Printf("\nConclusion: Bloom Filter added %v overhead. Cuckoo Filter added %v overhead.\n", overheadBf/time.Duration(len(idsToTest)), overheadCf/time.Duration(len(idsToTest)))
}

// --- Benchmark for Deletions (Cuckoo Only) ---
func benchmarkDeletions(cf *cuckoo.Filter, idsToTest [][]byte) {
	fmt.Println("\n-------------------------------------------------------------")
	log.Printf("--- Benchmark: Deletions (%d items) ---", len(idsToTest))
	fmt.Println("-------------------------------------------------------------")

	// Test 1: Deletion performance
	start := time.Now()
	for _, id := range idsToTest {
		cf.Delete(id)
	}
	duration := time.Since(start)
	fmt.Println("[Cuckoo Filter Deletion]")
	printMetrics(duration, len(idsToTest))

	// Test 2: Verification
	foundCount := 0
	for _, id := range idsToTest {
		if cf.Lookup(id) {
			foundCount++
		}
	}
	fmt.Printf("\nVerification: After deleting %d items, %d were still found in the filter.\n", len(idsToTest), foundCount)
	fmt.Println("Note: A standard Bloom Filter does not support deletion.")
}

// printMetrics is a helper function to display performance results.
func printMetrics(duration time.Duration, numOps int) {
	avg := duration / time.Duration(numOps)
	opsPerSec := float64(numOps) / duration.Seconds()
	
	fmt.Printf("  Total Time:       %v\n", duration)
	fmt.Printf("  Avg. Per Lookup:  %v\n", avg)
	fmt.Printf("  Ops/Second:       %.2f\n", opsPerSec)
}