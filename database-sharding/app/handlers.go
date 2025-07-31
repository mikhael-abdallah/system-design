package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type APIHandler struct {
	ShardManager *ShardManager
}

func (h *APIHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user.ID = uuid.New()

	shard := h.ShardManager.GetShardForID(user.ID)
	_, err := shard.InsertOne(context.Background(), user)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		log.Printf("Error in InsertOne: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *APIHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	shard := h.ShardManager.GetShardForID(id)
	var user User
	err = shard.FindOne(context.Background(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetUserByName is a costly operation in a system with ID-based sharding.
// It needs to query ALL shards.
func (h *APIHandler) GetUserByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var users []User
	var wg sync.WaitGroup
	var mu sync.Mutex
	allShards := h.ShardManager.GetAllShards()
	wg.Add(len(allShards))

	// Launch goroutines to query all shards in parallel.
	for _, shard := range allShards {
		go func(s *mongo.Collection) {
			defer wg.Done()
			cursor, err := s.Find(context.Background(), bson.M{"name": name})
			if err != nil {
				log.Printf("Error querying shard: %v", err)
				return
			}
			defer cursor.Close(context.Background())

			var shardUsers []User
			if err = cursor.All(context.Background(), &shardUsers); err != nil {
				log.Printf("Error decoding shard results: %v", err)
				return
			}

			// Use a mutex to add the results to the final list in a safe way.
			mu.Lock()
			users = append(users, shardUsers...)
			mu.Unlock()
		}(shard)
	}

	wg.Wait()

	if len(users) == 0 {
		http.Error(w, "No user found with that name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *APIHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find the correct shard.
	shard := h.ShardManager.GetShardForID(id)
	updateData := bson.M{
		"$set": bson.M{
			"name": updates["name"],
			"data": updates["data"],
		},
	}

	result, err := shard.UpdateOne(context.Background(), bson.M{"_id": id}, updateData)
	if err != nil || result.MatchedCount == 0 {
		http.Error(w, "User not found for update", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Find the correct shard and delete the user.
	shard := h.ShardManager.GetShardForID(id)
	result, err := shard.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil || result.DeletedCount == 0 {
		http.Error(w, "User not found for deletion", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}