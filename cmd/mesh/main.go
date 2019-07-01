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
	config     meshConfig
	siteConfig meshSiteConfig
)

const (
	queueCookie     = "akavpwr_VP1"
	queuePassCookie = "akavpau_VP1"
	sessionCookie   = "session.ID"
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

	json.Unmarshal(siteConfigBytes, &siteConfig)

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
}

func main() {

}

func createFrontendTask(SKU, regionCode string) *meshFrontendTask {
	if site, siteExists := siteConfig[regionCode]; siteExists {
		return &meshFrontendTask{
			SKU:            SKU,
			Site:           site,
			SiteCode:       regionCode,
			SessionCookies: make(map[string]*http.Cookie),
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
			SKU:      SKU,
			Site:     site,
			SiteCode: regionCode,
			Client: &http.Client{
				Timeout: 15 * time.Second,
			},
		}
	}

	return nil
}
