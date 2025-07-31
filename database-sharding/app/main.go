package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	shardManager, err := NewShardManager()
	if err != nil {
		log.Fatalf("Failed to initialize the Shard Manager: %v", err)
	}
	defer shardManager.Close()

	handler := &APIHandler{
		ShardManager: shardManager,
	}

	r := mux.NewRouter()

	r.HandleFunc("/users", handler.CreateUser).Methods("POST")
	r.HandleFunc("/users/{id}", handler.GetUserByID).Methods("GET")
	r.HandleFunc("/users/name/{name}", handler.GetUserByName).Methods("GET")
	r.HandleFunc("/users/{id}", handler.UpdateUser).Methods("PUT")
	r.HandleFunc("/users/{id}", handler.DeleteUser).Methods("DELETE")

	log.Println("Server started on port 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
}