package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/dchest/uniuri"
)

var (
	errTaskBanned = errors.New("Task is banned")

	errProductOOS       = errors.New("Product is out of stock")
	errProductNoSizes   = errors.New("Product has no available sizes")
	errProductNotLoaded = errors.New("Product not loaded")
)

func (t *endTask) Monitor() {
	log.Printf("[INFO] Starting task - %v", t.ProductSKU)
	t.SetProxy()

	for {
		sizeMap, err := t.GetSizes()

		if err != nil {
			switch err {
			case errProductOOS:
				log.Printf("[INFO] Product is out of stock, retrying - %v", t.ProductSKU)
				time.Sleep(750 * time.Millisecond)
				continue
			case errProductNoSizes:
				log.Printf("[INFO] Product has no available sizes, retrying - %v", t.ProductSKU)
				time.Sleep(750 * time.Millisecond)
				continue
			case errProductNotLoaded:
				log.Printf("[INFO] Product is not loaded, retrying - %v", t.ProductSKU)
				time.Sleep(1000 * time.Millisecond)
				continue
			case errTaskBanned:
				log.Printf("[WARN] Task is banned, retrying - %v", t.ProductSKU)
				t.SetProxy()
				time.Sleep(1000 * time.Millisecond)
				continue
			default:
				log.Printf("[ERROR] Unhandled Error - %v - %v", err.Error(), t.ProductSKU)
				t.SetProxy()
				time.Sleep(1500 * time.Millisecond)
				continue
			}
		}

		if len(sizeMap) == 0 {
			log.Printf("[INFO] Size map for product is empty, retrying - %v", t.ProductSKU)
			time.Sleep(750 * time.Millisecond)
			continue
		}

	}
}

func (t *endTask) SetProxy() {
	if len(config.Proxies) > 0 {
		proxy := config.Proxies[rand.Intn(len(config.Proxies))]

		proxyURL, err := url.Parse(proxy)

		if err != nil {
			log.Printf("Error %v - %v", t.ProductSKU, err.Error())
			log.Printf("[WARN] Running Proxyless - %v", t.ProductSKU)
			return
		}

		t.Client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		log.Printf("[INFO] Running Proxy (%v) - %v", proxyURL.String(), t.ProductSKU)
	} else {
		log.Printf("[WARN] Running Proxyless - %v", t.ProductSKU)
	}
}

func (t *endTask) GetSizes() (map[string]bool, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://admin2.endclothing.com/gb/rest/V1/end/products/sku/%v?/%v=%v", t.ProductSKU, uniuri.NewLen(8), uniuri.NewLen(8)), nil)

	if err != nil {
		return nil, err
	}

	resp, err := t.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		var product endProduct
		err = json.NewDecoder(resp.Body).Decode(&product)

		if err != nil {
			return nil, err
		}

		if t.ProductInfo != nil {
			prodInfo := &endProdInfo{
				Name:       product.Name,
				ProductURL: product.Link,
				Price:      fmt.Sprintf("£%v", product.Price),
			}

			if len(product.MediaGalleryEntries) > 0 {
				prodInfo.ImageURL = product.MediaGalleryEntries[0].File
			}

			t.ProductInfo = prodInfo
		}

		if product.InStock {
			sizesAvailable := false
			sizeMap := make(map[string]bool)
			for _, sizeOption := range product.Options {
				if sizeOption.AttributeID == "173" && sizeOption.Label == "Size" {
					if len(sizeOption.Values) > 0 {
						sizesAvailable = true
						for _, individualSize := range sizeOption.Values {
							sizeMap[individualSize.Label] = individualSize.InStock
						}
					}
				}
			}
			if sizesAvailable {
				return sizeMap, nil
			}
			return nil, errProductNoSizes
		}
		return nil, errProductOOS
	case 404:
		return nil, errProductNotLoaded
	case 403:
		return nil, errTaskBanned
	case 456:
		return nil, errTaskBanned
	default:
		return nil, fmt.Errorf("Invalid Status Code - %v", resp.StatusCode)
	}
}

func (t *endTask) CheckUpdate(sizeMap map[string]bool) {

}
