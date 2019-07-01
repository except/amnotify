package main

import (
	"net/http"
	"net/http/cookiejar"
	"time"
)

var (
	config     meshConfig
	siteConfig meshSiteConfig
)

func init() {

}

func main() {

}

func createFrontendTask(SKU, regionCode string) *meshFrontendTask {
	jar, err := cookiejar.New(nil)

	if err != nil {
		panic(err)
	}

	if site, siteExists := siteConfig[regionCode]; siteExists {
		return &meshFrontendTask{
			SKU:        SKU,
			Site:       site,
			SiteCode:   regionCode,
			SessionJar: jar,
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
