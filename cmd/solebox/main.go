package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	wg     sync.WaitGroup
	config sbxConfig
	client = &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

func init() {
	configFile, err := os.Open("config.json")

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
	}

	defer configFile.Close()
	configBytes, err := ioutil.ReadAll(configFile)

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
	}

	json.Unmarshal(configBytes, &config)

	log.Printf("[INFO] Loaded %v Webhooks - %v Products - %v Proxies", len(config.WebhookUrls), len(config.ProductUrls), len(config.ProxyArray))

}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	for _, productURL := range config.ProductUrls {
		wg.Add(1)

		go func(pURL string) {
			defer wg.Done()

			createProduct(pURL).launchMonitor()
		}(productURL)
	}

	wg.Wait()
}

func createProduct(prodURL string) *sbxProduct {
	return &sbxProduct{
		URL:              prodURL,
		SizeAvailability: make(map[string]bool),
		SizeMap:          make(map[string]string),
		FirstRun:         true,
		Client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}
