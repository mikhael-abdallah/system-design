package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// LeakyBucket represents the bucket structure
type LeakyBucket struct {
	capacity   int
	leakRate   int 
	queue      chan int
	leakTicker *time.Ticker
	mutex      sync.Mutex
}

// NewLeakyBucket creates and initializes a new leaky bucket
func NewLeakyBucket(capacity, leakRate int) *LeakyBucket {
	b := &LeakyBucket{
		capacity:   capacity,
		leakRate:   leakRate,
		queue:      make(chan int, capacity),
	}

	b.startLeaking()
	return b
}

// startLeaking begins the "leaking" process from the bucket
func (b *LeakyBucket) startLeaking() {
	b.leakTicker = time.NewTicker(time.Second / time.Duration(b.leakRate))
	go func() {
		for range b.leakTicker.C {
			select {
			case packetID := <-b.queue:
				fmt.Printf(" [LeakyBucket] Packet %d processed. Queue size: %d/%d\n", packetID, len(b.queue), b.capacity)
			default:
				// No packets in the queue, do nothing
			}
		}
	}()
}

// Stop stops the leaking process
func (b *LeakyBucket) Stop() {
	b.leakTicker.Stop()
}

// AddPacket adds a packet to the bucket's queue
func (b *LeakyBucket) AddPacket(packetID int) bool {
	select {
	case b.queue <- packetID:
		fmt.Printf(" [LeakyBucket] Packet %d added to queue. Queue size: %d/%d\n", packetID, len(b.queue), b.capacity)
		return true
	default:
		fmt.Printf(" [LeakyBucket] Packet %d discarded. Bucket queue is full!\n", packetID)
		return false
	}
}

// SimulateLeakyBucket simulates the algorithm
func SimulateLeakyBucket() {
	fmt.Println("--- Simulating Leaky Bucket ---")

	// Bucket capacity: 5, leak rate: 2 packets/second
	bucket := NewLeakyBucket(5, 2)
	defer bucket.Stop()

	// Simulate packet arrival in bursts
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

	time.Sleep(2 * time.Second) // Wait for the last packets to be processed
	fmt.Println("--- Leaky Bucket simulation finished ---")
}