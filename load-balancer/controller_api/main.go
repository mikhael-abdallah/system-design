package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	// Internal address of our load balancer (HAProxy)
	repositoryServiceUrl := "http://haproxy:8081/data"

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		log.Printf("Controller node '%s' received a request.", hostname)

		// Call the repository service through HAProxy
		resp, err := http.Get(repositoryServiceUrl)
		if err != nil {
			http.Error(w, "Error calling repository service: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		// Pass the repository service response to the final client
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		// Add a header to know which controller responded
		w.Header().Set("X-Controller-Node-ID", hostname)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Println("Controller server listening on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}