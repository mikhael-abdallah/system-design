package main

import (
	"hash"
	"hash/fnv"

	"github.com/spaolacci/murmur3"
)

// BloomFilter defines the data structure
type BloomFilter struct {
	m      uint64       // Size of the bit array
	k      uint64       // Number of hash functions
	bitset []uint64     // We use an array of uint64 for efficiency
	hash1  hash.Hash64  // First hash function
	hash2  hash.Hash64  // Second hash function
}

// NewBloomFilter creates and initializes a new Bloom Filter
func NewBloomFilter(m, k uint64) *BloomFilter {
	return &BloomFilter{
		m:      m,
		k:      k,
		bitset: make([]uint64, (m+63)/64), // Round up to the next multiple of 64
		hash1:  murmur3.New64(),
		hash2:  fnv.New64a(),
	}
}

// getHashes uses the double-hashing technique to generate k hashes
func (bf *BloomFilter) getHashes(data []byte) (uint64, uint64) {
	bf.hash1.Reset()
	bf.hash1.Write(data)
	h1 := bf.hash1.Sum64()

	bf.hash2.Reset()
	bf.hash2.Write(data)
	h2 := bf.hash2.Sum64()
	
	return h1, h2
}

// Add adds an item to the filter
func (bf *BloomFilter) Add(data []byte) {
	h1, h2 := bf.getHashes(data)
	for i := uint64(0); i < bf.k; i++ {
		// hash(i) = h1 + i * h2
		index := (h1 + i*h2) % bf.m
		// Set the bit at position 'index' to 1
		bf.bitset[index/64] |= (1 << (index % 64))
	}
}

// Test checks if an item "probably" is in the set
func (bf *BloomFilter) Test(data []byte) bool {
	h1, h2 := bf.getHashes(data)
	for i := uint64(0); i < bf.k; i++ {
		index := (h1 + i*h2) % bf.m
		// If we find a single bit 0, the item DEFINITELY is not in the set
		if (bf.bitset[index/64] & (1 << (index % 64))) == 0 {
			return false
		}
	}
	// If all bits are 1, the item PROBABLY is in the set
	return true
}