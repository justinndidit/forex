package util

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
)

func WriteJsonError(w http.ResponseWriter, status int, message string, details *string) {
	type ErrorResponse struct {
		Error   string  `json:"error"`
		Details *string `json:"details,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message, Details: details})
}

func WriteJsonSuccess(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

type FetchResult struct {
	URL  string
	Body []byte
	Err  error
}

func FetchData(url string, wg *sync.WaitGroup, resultChan chan<- FetchResult) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		resultChan <- FetchResult{URL: url, Err: fmt.Errorf("failed to fetch: %v", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		resultChan <- FetchResult{URL: url, Err: fmt.Errorf("bad status: %s", resp.Status)}
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resultChan <- FetchResult{URL: url, Err: fmt.Errorf("failed to read body: %v", err)}
		return
	}
	resultChan <- FetchResult{URL: url, Body: body}
}

func RandFloatRange() float64 {
	min := 1000.0
	max := 2000.0
	return min + rand.Float64()*(max-min)
}
