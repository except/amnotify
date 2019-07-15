package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	config endConfig
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

}

func createTask(productSKU string) *endTask {
	return &endTask{
		ProductSKU: productSKU,
		FirstRun:   true,
		SizeMap:    make(map[string]bool),
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}
