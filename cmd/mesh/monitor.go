package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
)

func (t *meshFrontendTask) beginMonitor() {

}

func (t *meshFrontendTask) setProxy() {
	if len(config.ProxyArray) > 0 {
		proxy := config.ProxyArray[rand.Intn(len(config.ProxyArray))]

		proxyURL, err := url.Parse(proxy)

		if err != nil {
			log.Printf("Error %v - %v", t.SKU, err.Error())
			log.Printf("[WARN] Running Proxyless (Frontend) - %v - %v", t.SKU, t.SiteCode)
			return
		}

		t.Client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		log.Printf("[INFO] Running Proxy (%v) - %v - %v", proxyURL.String(), t.SKU, t.SiteCode)
	} else {
		log.Printf("[WARN] Running Proxyless - %v - %v", t.SKU, t.SiteCode)
	}
}

func (t *meshFrontendTask) getSizes() (map[string]meshProductSKU, error) {
	return nil, nil
}

func (t *meshFrontendTask) addToWishlist() (bool, error) {
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v/wishlists/ajax", t.Site.SiteURL), nil)

	if err != nil {
		return false, err
	}

	resp, err := t.Client.Do(req)

	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:

	case 502:

	case 403:

	default:
		return false, fmt.Errorf("Invalid Status Code - %v - %v", t.SKU, t.SiteCode)
	}

	return false, nil
}

func (t *meshFrontendTask) setWishlistID() {

}

func (t *meshFrontendTask) detectQueue(cookies http.Cookie) bool {

	return false
}

func (t *meshFrontendTask) handleQueue() {

}

func (t *meshBackendTask) beginMonitor() {

}
