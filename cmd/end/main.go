package main

import (
	"crypto/tls"
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
	config endConfig
	client = &http.Client{
		Timeout: 20 * time.Second,
	}

	cookies = &endCookies{
		Map: make(map[string]time.Time),
	}

	cookieArray []string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

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

	cookieFile, err := os.Open("cookieArray.json")

	if err != nil {
		log.Printf("[ERROR] [COOKIE] %v", err.Error())
	}

	defer cookieFile.Close()

	cookieBytes, err := ioutil.ReadAll(cookieFile)

	if err != nil {
		log.Printf("[ERROR] [COOKIE] %v", err.Error())
	}

	err = json.Unmarshal(cookieBytes, &cookieArray)

	if err != nil {
		log.Printf("[ERROR] [COOKIE] %v", err.Error())
	}

	if err != nil {
		panic(err)
	}

	for _, cookie := range cookieArray {
		cookies.Map[cookie] = time.Now()
	}

	return
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
		ProductSKU:   productSKU,
		FirstRun:     true,
		SizeMap:      make(map[string]bool),
		IndexMap:     make(map[string]string),
		RequestCount: 0,
		Client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}
