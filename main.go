package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

func main() {
	// Command-line flags
	ratePtr := flag.Int("rate", 1, "Number of requests per second")
	delayPtr := flag.Int("delay", 1000, "Time interval between two requests (in milliseconds)")
	reflectPtr := flag.String("reflect", "swagnito", "Reflection parameter value")

	// Help menu
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] file_path\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check if the file path is provided as a command-line argument
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	filePath := flag.Args()[0]

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open the file: %v", err)
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)

	// Channel to control the rate and delay of requests
	requestsCh := make(chan string, *ratePtr)
	delayCh := time.Tick(time.Duration(*delayPtr) * time.Millisecond)

	// Wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Start a goroutine to process the requests
	go processRequests(requestsCh, *reflectPtr, &wg)

	// Read the file and send requests
	for scanner.Scan() {
		urlString := scanner.Text()

		// Send the URL to the requests channel
		requestsCh <- urlString

		// Wait for the specified delay interval
		<-delayCh
	}

	// Close the requests channel to signal the processing goroutine to exit
	close(requestsCh)

	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read the file: %v", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()
}

// Goroutine to process the requests
func processRequests(requestsCh <-chan string, reflectValue string, wg *sync.WaitGroup) {
	// Wait group to track the processing goroutines
	var requestWg sync.WaitGroup

	// Loop through the requests channel
	for request := range requestsCh {
		// Increment the wait group for each request
		requestWg.Add(1)

		// Process the request in a separate goroutine
		go func(req string) {
			defer requestWg.Done()

			// Parse the URL
			u, err := url.Parse(req)
			if err != nil {
				log.Printf("Failed to parse URL '%s': %v", req, err)
				return
			}

			// Get the query parameters
			queryParams := u.Query()

			// Map to store the modified query parameter values
			modifiedParams := make(map[string]string)

			// Replace all query parameter values with the reflection value
			for key := range queryParams {
				modifiedParams[key] = reflectValue
				queryParams.Set(key, reflectValue)
			}

			// Update the URL with modified query parameters
			u.RawQuery = queryParams.Encode()

			// Send an HTTP GET request to the modified URL
			resp, err := http.Get(u.String())
			if err != nil {
				log.Printf("Failed to send request to URL '%s': %v", u.String(), err)
				return
			}
			defer resp.Body.Close()

			// Read the response body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Failed to read response body for URL '%s': %v", u.String(), err)
				return
			}

			// Check if the response body contains the reflection value
			if strings.Contains(string(body), reflectValue) {
				// Print the modified URL with replaced query parameters
				modifiedURL := replaceParamsInURL(u.String(), modifiedParams)
				fmt.Printf("%s [reflected]%s\n", colorizeText(modifiedURL, colorRed), colorReset)
			}
		}(request)
	}

	// Wait for all requests to finish processing
	requestWg.Wait()

	// Signal the main goroutine that all processing is complete
	wg.Done()
}

// Helper function to replace query parameter values in URL
func replaceParamsInURL(urlStr string, params map[string]string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	queryParams := u.Query()
	for key, value := range params {
		queryParams.Set(key, value)
	}

	u.RawQuery = queryParams.Encode()
	return u.String()
}

// Helper function to colorize text
func colorizeText(text, color string) string {
	return fmt.Sprintf("%s%s%s", color, text, colorReset)
}
