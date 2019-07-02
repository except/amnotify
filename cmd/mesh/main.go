package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	wg         sync.WaitGroup
	config     meshConfig
	siteConfig meshSiteConfig

	client = &http.Client{
		Timeout: 15 * time.Second,
	}
)

const (
	queueCookie     = "akavpwr_VP1"
	queuePassCookie = "akavpau_VP1"
	sessionCookie   = "session.ID"
	itemInStock     = "IN STOCK"
	itemOutOfStock  = "OUT OF STOCK"
)

func init() {
	siteConfigFile, err := os.Open("regions.json")

	if err != nil {
		log.Printf("[ERROR] [SITE CONFIG] %v", err.Error())
	}

	defer siteConfigFile.Close()
	siteConfigBytes, err := ioutil.ReadAll(siteConfigFile)

	if err != nil {
		log.Printf("[ERROR] [SITE CONFIG] %v", err.Error())
	}

	err = json.Unmarshal(siteConfigBytes, &siteConfig)

	if err != nil {
		log.Printf("[ERROR] [SITE CONFIG] %v", err.Error())
	}

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
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	for _, task := range config.Tasks {
		for _, regionCode := range task.Sites {
			wg.Add(1)

			go func(SKU, regionCode string) {
				defer wg.Done()
				task := createFrontendTask(SKU, regionCode)
				if task != nil {
					task.Monitor()
				}
			}(task.SKU, regionCode)

			// go func(SKU, regionCode string) {
			// 	defer wg.Done()
			// 	task := createBackendTask(SKU, regionCode)
			// 	if task != nil {
			// 		task.Monitor()
			// 	}
			// }(task.SKU, regionCode)
		}
	}

	wg.Wait()
}

func createFrontendTask(SKU, regionCode string) *meshFrontendTask {
	if site, siteExists := siteConfig[regionCode]; siteExists {
		return &meshFrontendTask{
			SKU:            SKU,
			Site:           site,
			SiteCode:       regionCode,
			SessionCookies: make(map[string]*http.Cookie),
			ProductSKUMap:  make(map[string]meshProductSKU),
			Client: &http.Client{
				Timeout: 15 * time.Second,
			},
		}
	}

	return nil
}

func createBackendTask(SKU, regionCode string) *meshBackendTask {
	if site, siteExists := siteConfig[regionCode]; siteExists {
		return &meshBackendTask{
			SKU:           SKU,
			Site:          site,
			SiteCode:      regionCode,
			ProductSKUMap: make(map[string]meshProductSKU),
			Client: &http.Client{
				Timeout: 15 * time.Second,
			},
		}
	}

	return nil
}
