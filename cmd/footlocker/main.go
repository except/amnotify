package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var (
	wg     sync.WaitGroup
	config ftlConfig

	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

func init() {
	proxyURL, _ := url.Parse("http://81.92.194.200:8800")
	client.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// Update
	// Update
	// Update

	configFile, err := os.Open("config.json")

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
		return
	}

	defer configFile.Close()
	configBytes, err := ioutil.ReadAll(configFile)

	if err != nil {
		log.Printf("[ERROR] [CONFIG] %v", err.Error())
		return
	}

	json.Unmarshal(configBytes, &config)

	log.Printf("[INFO] Loaded %v Products", len(config.SKUArray))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	for _, product := range config.SKUArray {
		for _, region := range product.Regions {
			wg.Add(1)

			go func(productSKU, region string) {
				defer wg.Done()

				createTask(productSKU, region).beginMonitor()
			}(product.SKU, region)
		}
	}

	wg.Wait()
}

func createTask(productSKU, region string) *ftlTask {
	selectedRegion, regionExists := config.Regions[region]
	if regionExists {
		return &ftlTask{
			SKU:        productSKU,
			Region:     selectedRegion,
			RegionName: region,
			FirstRun:   true,
			Inventory:  make(map[string]ftlSize),
		}
	}

	return nil
}
