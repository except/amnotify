package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
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

			if t.FirstRun {
				t.FirstRun = false
			}

			switch err {
			case errProductOOS:
				log.Printf("[INFO] Product is out of stock, retrying - %v", t.ProductSKU)
				time.Sleep(1500 * time.Millisecond)
				continue
			case errProductNoSizes:
				log.Printf("[INFO] Product has no available sizes, retrying - %v", t.ProductSKU)
				time.Sleep(1500 * time.Millisecond)
				continue
			case errProductNotLoaded:
				log.Printf("[INFO] Product is not loaded, retrying - %v", t.ProductSKU)
				time.Sleep(1500 * time.Millisecond)
				continue
			case errTaskBanned:
				log.Printf("[WARN] Task is banned, retrying - %v", t.ProductSKU)
				t.SetProxy()
				time.Sleep(2500 * time.Millisecond)
				continue
			default:
				log.Printf("[ERROR] Unhandled Error - %v - %v", err.Error(), t.ProductSKU)
				t.SetProxy()
				time.Sleep(2500 * time.Millisecond)
				continue
			}
		}

		if len(sizeMap) == 0 {
			log.Printf("[INFO] Size map for product is empty, retrying - %v", t.ProductSKU)
			time.Sleep(1500 * time.Millisecond)
			continue
		}

		log.Printf("[INFO] Gathered size map - %v", t.ProductSKU)
		t.CheckUpdate(sizeMap)
		time.Sleep(1500 * time.Millisecond)
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
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://www.endclothing.com/gb/rest/V1/end/products/sku/%v?/%v=%v", t.ProductSKU, uniuri.NewLen(16), uniuri.NewLen(16)), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")

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

		if t.ProductInfo == nil {
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
	updateAvailable := false
	for size, stockAvailable := range sizeMap {
		if sizeInstock, sizeExists := t.SizeMap[size]; sizeExists {
			if !sizeInstock && stockAvailable {
				updateAvailable = true
			}
		} else if stockAvailable {
			updateAvailable = true
		}
	}

	t.SizeMap = sizeMap

	if updateAvailable {
		if t.FirstRun {
			log.Printf("[INFO] Ignoring first run update - %v", t.ProductSKU)
		} else {
			log.Printf("[INFO] Update available - %v", t.ProductSKU)
			for _, webhookURL := range config.WebhookUrls {
				go t.SendUpdate(webhookURL)
			}
		}
	} else {
		log.Printf("[INFO] No update available - %v", t.ProductSKU)
	}
}

func (t *endTask) SendUpdate(webhookURL string) {
	webhook := &discordWebhook{}

	webhookEmbed := discordEmbed{
		Title: t.ProductInfo.Name,
		URL:   fmt.Sprintf("%v?/%v=%v", t.ProductInfo.ProductURL, uniuri.NewLen(4), uniuri.NewLen(4)),
		Color: 16721733,
	}

	webhookEmbed.Thumbnail = discordEmbedThumbnail{
		URL: t.ProductInfo.ImageURL,
	}

	webhookEmbed.Fields = append(webhookEmbed.Fields, discordEmbedField{
		Name:   "Price",
		Value:  t.ProductInfo.Price,
		Inline: true,
	})

	webhookEmbed.Fields = append(webhookEmbed.Fields, discordEmbedField{
		Name:   "Product SKU",
		Value:  strings.ToUpper(t.ProductSKU),
		Inline: true,
	})

	var sizeFloats []float64
	var sortedSizes []string

	for size, sizeAvail := range t.SizeMap {
		if sizeAvail {
			sizeFloat, err := strconv.ParseFloat(strings.Replace(size, "UK ", "", -1), 64)
			if err != nil {
				fmt.Println(err)
				continue
			}

			sizeFloats = append(sizeFloats, sizeFloat)
		}
	}

	sort.Float64s(sizeFloats)

	for _, sizeFloat := range sizeFloats {
		sortedSizes = append(sortedSizes, fmt.Sprintf("UK %g", sizeFloat))
	}

	webhookEmbed.Fields = append(webhookEmbed.Fields, discordEmbedField{
		Name:   "Size Availability",
		Value:  strings.Join(sortedSizes, "\n"),
		Inline: false,
	})

	webhookEmbed.Footer = discordEmbedFooter{
		Text:    fmt.Sprintf("AMNotify | END • %v", time.Now().Format("15:04:05.000")),
		IconURL: "https://i.imgur.com/vv2dyGR.png",
	}

	webhook.Embeds = append(webhook.Embeds, webhookEmbed)

	webhookPayload, err := json.Marshal(webhook)

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", t.ProductSKU, err.Error())
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(webhookPayload))

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", t.ProductSKU, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", t.ProductSKU, err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		log.Printf("[SUCCESS] Webhook sent - %v", t.ProductSKU)
	} else if resp.StatusCode == 429 {
		log.Printf("[WARN] Retrying, webhook ratelimit - %v", t.ProductSKU)
		time.Sleep(5 * time.Second)
		t.SendUpdate(webhookURL)
	} else {
		log.Printf("[WARN] Invalid Status - %v - %v", t.ProductSKU, resp.Status)
	}
}
