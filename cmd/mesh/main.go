package main

import (
	"net/http"
	"time"
)

var (
	config     meshConfig
	siteConfig meshSiteConfig
)

const (
	queueCookie   = "QueueCookie"
	sessionCookie = "SessionCookie"
)

func init() {

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
