package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	numShards = 4
)

// ShardManager manages the connections with all MongoDB shards
type ShardManager struct {
	Clients []*mongo.Client
	Shards  []*mongo.Collection
}

// NewShardManager creates and tests the connections with all MongoDB shards
func NewShardManager() (*ShardManager, error) {
	manager := &ShardManager{
		Clients: make([]*mongo.Client, numShards),
		Shards:  make([]*mongo.Collection, numShards),
	}

	for i := 0; i < numShards; i++ {
		// The service name in Docker Compose will be 'mongo-shard-0', 'mongo-shard-1', etc.
		uri := fmt.Sprintf("mongodb://mongo-shard-%d:27017", i)
		client, err := mongo.NewClient(options.Client().ApplyURI(uri))
		if err != nil {
			return nil, fmt.Errorf("error creating client for shard %d: %w", i, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("error connecting to shard %d: %w", i, err)
		}

		// Test the connection
		if err := client.Ping(ctx, nil); err != nil {
			return nil, fmt.Errorf("ping failed for shard %d: %w", i, err)
		}

		log.Printf("Connected successfully to Shard %d", i)
		manager.Clients[i] = client
		manager.Shards[i] = client.Database("userdb").Collection("users")
	}

	return manager, nil
}

// getShardIndex calculates in which shard a given ID should be.
func getShardIndex(id uuid.UUID) int {
	// We use an FNV-1a hasher, which is fast and offers good distribution.
	hasher := fnv.New64a()
	hasher.Write(id[:])
	hash := hasher.Sum64()

	// The modulo operator gives us the shard index (0, 1, 2 or 3).
	return int(hash % uint64(numShards))
}

func (sm *ShardManager) GetShardForID(id uuid.UUID) *mongo.Collection {
	index := getShardIndex(id)
	return sm.Shards[index]
}

func (sm *ShardManager) GetAllShards() []*mongo.Collection {
	return sm.Shards
}

func (sm *ShardManager) Close() {
	for i, client := range sm.Clients {
		if client != nil {
			if err := client.Disconnect(context.Background()); err != nil {
				log.Printf("Error disconnecting from shard %d: %v", i, err)
			}
		}
	}
}