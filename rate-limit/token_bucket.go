package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// TokenBucket represents the token bucket structure
type TokenBucket struct {
	capacity      int
	tokens        int
	tokenRate     int 
	lastRefill    time.Time
	mutex         sync.Mutex
	packetQueue   chan int
}

// NewTokenBucket creates and initializes a new token bucket
func NewTokenBucket(capacity, tokenRate, queueCapacity int) *TokenBucket {
	tb := &TokenBucket{
		capacity:   capacity,
		tokens:     capacity, // Start with a full bucket
		tokenRate:  tokenRate,
		lastRefill: time.Now(),
		packetQueue: make(chan int, queueCapacity),
	}
	
	// Start a worker to process packets when tokens are available
	go tb.processor()
	return tb
}

// refill adds tokens to the bucket based on time
func (b *TokenBucket) refill() {
	now := time.Now()
	// Calculate how many tokens should have been added since the last refill
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(b.tokenRate))

	if tokensToAdd > 0 {
		b.tokens = min(b.tokens + tokensToAdd, b.capacity)
		b.lastRefill = now
	}
}

// processor handles taking packets from the queue and tokens from the bucket
func (b *TokenBucket) processor() {
	ticker := time.NewTicker(time.Second / time.Duration(b.tokenRate))
	defer ticker.Stop()
	
	for range ticker.C {
		b.mutex.Lock()
		b.refill()
		b.mutex.Unlock()

		select {
		case packetID := <-b.packetQueue:
			b.mutex.Lock()
			if b.tokens > 0 {
				b.tokens--
				fmt.Printf(" [TokenBucket] Packet %d sent! Tokens remaining: %d/%d\n", packetID, b.tokens, b.capacity)
			}
			b.mutex.Unlock()
		default:
			// No packets in the queue, do nothing
		}
	}
}

// AddPacket adds a packet to the token bucket's queue
func (b *TokenBucket) AddPacket(packetID int) bool {
	select {
	case b.packetQueue <- packetID:
		fmt.Printf(" [TokenBucket] Packet %d added to queue. Queue size: %d/%d\n", packetID, len(b.packetQueue), cap(b.packetQueue))
		return true
	default:
		fmt.Printf(" [TokenBucket] Packet %d discarded. Queue is full!\n", packetID)
		return false
	}
}

// SimulateTokenBucket simulates the algorithm
func SimulateTokenBucket() {
	fmt.Println("--- Simulating Token Bucket ---")

	// Bucket capacity: 5 tokens, token rate: 2/second, queue capacity: 10
	bucket := NewTokenBucket(5, 2, 10)

	// Simulate packet arrival
	for i := 0; i < 20; i++ {
		// A burst of 1 to 4 packets every 500ms
		if i%2 == 0 {
			numPackets := rand.Intn(4) + 1
			for j := 0; j < numPackets; j++ {
				bucket.AddPacket(i*10 + j)
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(4 * time.Second) // Wait for the processor to finish
	fmt.Println("--- Token Bucket simulation finished ---")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	SimulateLeakyBucket()
	fmt.Println()
	SimulateTokenBucket()
}