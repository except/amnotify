package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	wg     sync.WaitGroup
	config endConfig

	client = &http.Client{
		Timeout: 20 * time.Second,
	}
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	configFile, err := os.Open("config.json")

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
	}

	defer configFile.Close()

	configBytes, err := ioutil.ReadAll(configFile)

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
	}

	err = json.Unmarshal(configBytes, &config)

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
	}

	if err != nil {
		panic(err)
	}
}

func main() {
	for _, productSKU := range config.ProductSKUs {
		wg.Add(1)

		go func(productSKU string) {
			defer wg.Done()

			createTask(productSKU).Monitor()
		}(productSKU)
	}
	wg.Wait()
}

func createTask(productSKU string) *endTask {
	return &endTask{
		ProductSKU: productSKU,
		FirstRun:   true,
		SizeMap:    make(map[string]bool),
		Client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}
