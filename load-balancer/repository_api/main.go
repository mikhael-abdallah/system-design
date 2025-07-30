package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Get the connection string from the environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not defined")
	}

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Handler for the request
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		log.Printf("Repository node '%s' received a request.", hostname)

		// Get a random message from the database
		var message string
		err := db.QueryRow("SELECT message FROM messages ORDER BY RANDOM() LIMIT 1").Scan(&message)
		if err != nil {
			http.Error(w, "Error querying the database: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// await random time between 0 and 10 seconds
		waitTime := time.Duration(rand.Intn(10000)) * time.Millisecond
		log.Printf("Repository node '%s' waiting for %s", hostname, waitTime)
		time.Sleep(waitTime)

		// Respond with JSON
		response := map[string]string{
			"data_message":      message,
			"repository_node_id": hostname,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Println("Repository server listening on port 8001...")
	log.Fatal(http.ListenAndServe(":8001", nil))
}