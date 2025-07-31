package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Data string    `json:"data"`
}

const (
	apiURL    = "http://localhost:8080"
	numUsers  = 1000
	numShards = 4
)

var (
	green  = color.New(color.FgGreen).PrintlnFunc()
	blue   = color.New(color.FgBlue).PrintlnFunc()
	yellow = color.New(color.FgYellow).PrintlnFunc()
	red    = color.New(color.FgRed).PrintlnFunc()
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// --- 1. Insert 1000 Users in Parallel ---
func insertUsers() {
	blue("--- 1. Inserting", numUsers, "users in parallel ---")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) 
	repeatingNames := []string{"John Doe", "Jane Smith", "Peter Jones"}

	for i := 1; i <= numUsers; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(i int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			var name string
			if i%100 == 0 {
				name = repeatingNames[(i/100-1)%3]
			} else {
				name = fmt.Sprintf("User %d", i)
			}
			data := fmt.Sprintf("Random data for %s", name)

			payload := map[string]string{"name": name, "data": data}
			jsonData, _ := json.Marshal(payload)

			resp, err := httpClient.Post(apiURL+"/users", "application/json", bytes.NewBuffer(jsonData))
			if err != nil || resp.StatusCode != http.StatusCreated {
				log.Printf("Error inserting user %d: %v", i, err)
				return
			}
			defer resp.Body.Close()
			
			fmt.Print(".")
		}(i)
	}

	wg.Wait()
	fmt.Println()
	green(numUsers, "users inserted successfully.")
}

// --- 2. Count Users in Each Shard ---
func countShards() {
	blue("\n--- 2. Checking the data distribution in the shards ---")
	totalCount := 0

	for i := 0; i < numShards; i++ {	
		uri := fmt.Sprintf("mongodb://localhost:%d", 27017+i)
		client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
		if err != nil {
			red("Error connecting to shard", i, ":", err)
			continue
		}
		defer client.Disconnect(context.Background())

		collection := client.Database("userdb").Collection("users")
		count, err := collection.CountDocuments(context.Background(), map[string]interface{}{})
		if err != nil {
			red("Error counting documents in shard", i, ":", err)
			continue
		}
		
		yellow(fmt.Sprintf("Shard %d: %d users", i, count))
		totalCount += int(count)
	}
	green("Total users in the shards:", totalCount)
	yellow("Note: The distribution will not be perfect, but should be close due to the hash.")
}

// --- 3. Testing the CRUD functionalities ---
func testCRUD() {
	blue("\n--- 3. Testing the CRUD functionalities ---")

	// a. Create
	yellow("\n-> Testing POST /users")
	createPayload := map[string]string{"name": "Test CRUD", "data": "initial data"}
	jsonData, _ := json.Marshal(createPayload)
	resp, _ := httpClient.Post(apiURL+"/users", "application/json", bytes.NewBuffer(jsonData))
	
	var testUser User
	json.NewDecoder(resp.Body).Decode(&testUser)
	resp.Body.Close()
	green("Test user created with ID:", testUser.ID.String())

	// b. Get by ID
	yellow("\n-> Testing GET /users/{id}")
	resp, _ = httpClient.Get(apiURL + "/users/" + testUser.ID.String())
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	green("Get by ID successful. Response:", string(bodyBytes))

	// c. Update
	yellow("\n-> Testing PUT /users/{id}")
	updatePayload := map[string]string{"name": "Teste CRUD Atualizado", "data": "dados atualizados"}
	jsonData, _ = json.Marshal(updatePayload)
	req, _ := http.NewRequest(http.MethodPut, apiURL+"/users/"+testUser.ID.String(), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	httpClient.Do(req)
	green("Update request sent. Checking...")
	resp, _ = httpClient.Get(apiURL + "/users/" + testUser.ID.String())
	bodyBytes, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	green("Response after update:", string(bodyBytes))

	// d. Get by name (Scatter-Gather)
	yellow("\n-> Testing GET /users/name/{name} (Scatter-Gather)")
	resp, _ = httpClient.Get(apiURL + "/users/name/John%20Doe")
	var multiUserResponse []User
	json.NewDecoder(resp.Body).Decode(&multiUserResponse)
	resp.Body.Close()
	green(fmt.Sprintf("Get by name 'John Doe' found %d users (expected > 1).", len(multiUserResponse)))

	// e. Delete
	yellow("\n-> Testing DELETE /users/{id}")
	req, _ = http.NewRequest(http.MethodDelete, apiURL+"/users/"+testUser.ID.String(), nil)
	resp, _ = httpClient.Do(req)
	green(fmt.Sprintf("User deleted. Status Code: %d (expected 204)", resp.StatusCode))
}

// --- 4. Testing Failure Cases ---
func testFailures() {
	blue("\n--- 4. Testing failure cases (non-existent IDs) ---")
	nonExistentID := uuid.New()
	yellow("Using non-existent ID for tests:", nonExistentID.String())

	// GET
	resp, _ := httpClient.Get(apiURL + "/users/" + nonExistentID.String())
	fmt.Printf("-> Testing GET of non-existent ID (expected 404): %d ", resp.StatusCode)
	if resp.StatusCode == http.StatusNotFound { green("OK") } else { red("FALHOU") }

	// PUT
	req, _ := http.NewRequest(http.MethodPut, apiURL+"/users/"+nonExistentID.String(), bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = httpClient.Do(req)
	fmt.Printf("-> Testing PUT of non-existent ID (expected 404): %d ", resp.StatusCode)
	if resp.StatusCode == http.StatusNotFound { green("OK") } else { red("FALHOU") }

	// DELETE
	req, _ = http.NewRequest(http.MethodDelete, apiURL+"/users/"+nonExistentID.String(), nil)
	resp, _ = httpClient.Do(req)
	fmt.Printf("-> Testing DELETE of non-existent ID (expected 404): %d ", resp.StatusCode)
	if resp.StatusCode == http.StatusNotFound { green("OK") } else { red("FALHOU") }
}

func main() {
	insertUsers()
	countShards()
	testCRUD()
	testFailures()
	green("\n--- All tests completed! ---")
}