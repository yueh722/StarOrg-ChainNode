package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HttpGet performs an HTTP GET request and returns the response body.
func HttpGet(url string) ([]byte, error) {
	// Create a new HTTP client with a timeout.
	client := &http.Client{Timeout: 10 * time.Second}

	// Perform the GET request.
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for non-200 status codes.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send http request, code: %d, err: %v", resp.StatusCode, err)
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// HttpPost performs an HTTP POST request and returns the response body.
func HttpPost(url string, arg any) (interface{}, error) {
	// Create a new HTTP client with a timeout.
	client := &http.Client{Timeout: 10 * time.Second}

	// Convert the arguments to JSON.
	body, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	// Perform the POST request with JSON body.
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for non-200 status codes.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send http request, code: %d, err: %v", resp.StatusCode, err)
	}

	// Read the response body.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Attempt to decode the body as JSON (assuming the response is JSON).
	var result interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		// If it's not JSON, return the raw body.
		return string(respBody), nil
	}

	return result, nil
}
