package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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

	if err != nil {
		panic(err)
	}
}

func main() {
	log.Printf("Loaded | Proxies [%v] - Tasks [%v]", len(config.ProxyArray), len(config.Tasks))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	for _, proxy := range config.ProxyArray {
		testProxy(proxy)
	}

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
		}
	}

	wg.Wait()
}

func testProxy(proxyStr string) {
	testClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	proxyURL, err := url.Parse(proxyStr)

	if err != nil {
		log.Printf("[WARN] Proxy FAIL - %v", proxyStr)
		return
	}

	testClient.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	req, err := http.NewRequest(http.MethodGet, siteConfig["FP_UK"].SiteURL, nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-gb")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1 Mobile/15E148 Safari/604.1")

	if err != nil {
		log.Printf("[WARN] Proxy FAIL - %v", proxyStr)
		return
	}

	resp, err := testClient.Do(req)

	if err != nil {
		log.Printf("[WARN] Proxy FAIL - %v", proxyStr)
		return
	}

	if resp.StatusCode == 200 {
		log.Printf("[SUCCESS] Proxy PASS - %v", proxyStr)
		return
	}

	log.Printf("[WARN] Proxy FAIL - %v - %v", resp.StatusCode, proxyStr)
	return
}

func createFrontendTask(SKU, regionCode string) *meshFrontendTask {
	if site, siteExists := siteConfig[regionCode]; siteExists {
		return &meshFrontendTask{
			SKU:            SKU,
			FirstRun:       true,
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
