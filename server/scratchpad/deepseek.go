package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func getDeepSeekAnswer(input string, deepseek_key string) (string, error) {
	url := "https://api.deepseek.com/chat/completions"

	requestBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": input},
		},
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return "", err
	}

	// Create a new request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return "", err
	}

	// Set content type and authorization header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+deepseek_key)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", resp.StatusCode)
		return "", err
	}

	// Read the response body (optional)
	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return "", err
	}

	// Print the response (for debugging purposes)
	fmt.Printf("Response: %+v\n", responseBody)
	return "", nil

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	deepseek_key := os.Getenv("DEEPSEEK_KEY")

	getDeepSeekAnswer("Hello world", deepseek_key)
}
