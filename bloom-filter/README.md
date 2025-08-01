# Bloom Filter Performance Demonstration with Go and PostgreSQL

This project provides a practical demonstration of a **Bloom Filter**, a space-efficient probabilistic data structure used to test whether an element is a member of a set.

The goal is to showcase the significant performance benefits of using an in-memory Bloom Filter to avoid expensive database lookups for items that do not exist in a massive dataset. The application seeds a PostgreSQL database with **20 million records**, warms up a Bloom Filter, and then runs a series of benchmarks to compare lookup times.

## What is a Bloom Filter?

A Bloom Filter can tell you one of two things about an item:
1.  The item is **definitely not** in the set.
2.  The item is **probably** in the set.

This means a Bloom Filter **never has false negatives** (if an item was added, it will always be found) but has a small, configurable probability of **false positives** (it might say an item is present when it's not). This trade-off allows it to be incredibly fast and memory-efficient, making it ideal as a pre-filter for more resource-intensive checks like database queries.

## Implementation Details

### Filter Sizing
To handle a large dataset, the Bloom Filter's parameters must be calculated based on the number of items and the desired false positive rate. For this project, with a target of **n = 20 million items** and a false positive probability of **p = 1% (0.01)**, the optimal parameters are:

-   **Bit Array Size (`m`):** ~192 million bits (**~23 MB**)
-   **Hash Functions (`k`):** **7**

It's remarkable that we can represent the existence of 20 million unique items in just **23 MB** of RAM.

### Hashing Technique
To generate 7 distinct hashes efficiently, this project uses a "double-hashing" technique. Two fast, independent hash functions (Murmur3 and FNV-1a) are used to create a sequence of hashes for any given item, avoiding the overhead of initializing 7 separate hashers.

### Database & Warm-up
On its first run, the application performs two time-consuming tasks:
1.  **Database Seeding:** It populates the PostgreSQL database with 20 million user records using the efficient `COPY FROM` command.
2.  **Filter Warm-up:** It reads all 20 million UUIDs from the database and adds them to the in-memory Bloom Filter.

This process ensures that subsequent runs will have a fully populated database and a ready-to-use filter.

## Prerequisites

* Docker & Docker Compose
* Go

## How to Run

**IMPORTANT:** The first time you run this project, it will seed the database and warm up the filter. This is a one-time process that can be very time-consuming (potentially 15-30 minutes or more depending on your hardware). Subsequent runs will be much faster.

1.  **Initialize Go Modules:**
    ```bash
    cd app
    go mod init go-bloom-filter/app
    go mod tidy
    cd ..
    ```

2.  **Start the Application & Benchmarks:**
    ```bash
    docker compose up --build
    ```
    The application will start, seed the database if necessary, warm up the filter, and then automatically run the benchmarks, printing the results to the console.

## Benchmark Analysis

The tests were conducted by performing 100,000 lookups for non-existent keys and 100,000 lookups for existing keys. The results clearly demonstrate the effectiveness of the Bloom Filter.

### Test 1: Non-Existent User Lookups
This test measures the time it takes to determine that a user is *not* in the database.

| Method                | Total Time     | Avg. Per Lookup | Throughput (Ops/Sec) |
| --------------------- | -------------- | --------------- | -------------------- |
| **With Bloom Filter** | **~23.7 ms** | **~237 ns** | **~4,210,269** |
| **Database Only** | **~5.88 s** | **~58.8 µs** | **~16,996** |

**Conclusion:** The Bloom Filter was **~300 times faster** at identifying non-existent items. It reduces the lookup time from microseconds (µs) to nanoseconds (ns) by completely avoiding network and disk I/O. Furthermore, the observed false positive rate was only **1%**, exactly our target.

---

### Test 2: Existing User Lookups
This test measures the performance for items that *are* in the database, showing the minor overhead added by checking the filter first.

| Method                    | Total Time     | Avg. Per Lookup | Throughput (Ops/Sec) |
| ------------------------- | -------------- | --------------- | -------------------- |
| **Bloom Filter + DB** | **~5.80 s** | **~58.0 µs** | **~17,227** |
| **Database Only** | **~5.58 s** | **~55.8 µs** | **~17,919** |

**Conclusion:** The overhead for checking an existing item against the Bloom Filter before querying the database is **negligible**, adding **~2.2 microseconds (µs)** per operation. Adding a **4% increase** in the time. This is a tiny price to pay for the massive gains seen with non-existent items.


## Overall Conclusion

The benchmark results clearly illustrate the classic trade-off of a Bloom Filter. For the small price of a ~4% performance overhead on successful lookups and a tiny, predictable false positive rate, the system gains a ~300x performance improvement for lookups of non-existent items.

Therefore, for any application that must efficiently check for the existence of items and anticipates a significant percentage of lookups for things that are not in the set—such as checking for username availability, validating single-use coupon codes, or preventing duplicate article submissions—implementing a Bloom Filter is an extremely effective strategy. It dramatically reduces unnecessary load on the primary database and provides a much faster response time for "miss" cases.