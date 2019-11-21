package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dchest/uniuri"
)

var (
	errTaskBanned = errors.New("Task is banned")

	errProductOOS       = errors.New("Product is out of stock")
	errProductNoSizes   = errors.New("Product has no available sizes")
	errProductNotLoaded = errors.New("Product not loaded")

	errChallengeNoPath = errors.New("Failed to complete challenge - Path not found")
	errChallengeFailed = errors.New("Failed to complete challenge")

	siteURL, siteURLErr = url.Parse("https://api2.endclothing.com")
)

func (t *endTask) Monitor() {
	if siteURLErr != nil {
		panic(siteURLErr)
	}

	log.Printf("[INFO] Starting task - %v", t.ProductSKU)

	for {
		productURL := fmt.Sprintf("https://distilnetworks.endservices.info/gb/rest/V1/end/products/sku/%v/", t.ProductSKU /*uniuri.NewLen(16), uniuri.NewLen(16) */)
		t.PurgeURL(productURL)
		sizeMap, err := t.GetSizes(productURL)

		if err != nil {
			switch err {
			case errProductOOS:
				if t.FirstRun {
					t.FirstRun = false
					for size := range t.SizeMap {
						t.SizeMap[size] = false
					}
				}
				log.Printf("[INFO] Product is out of stock, retrying - %v", t.ProductSKU)
				// time.Sleep(1500 * time.Millisecond)
				continue
			case errProductNoSizes:
				if t.FirstRun {
					t.FirstRun = false
				}
				log.Printf("[INFO] Product has no available sizes, retrying - %v", t.ProductSKU)
				// time.Sleep(1500 * time.Millisecond)
				continue
			case errProductNotLoaded:
				if t.FirstRun {
					t.FirstRun = false
				}
				log.Printf("[INFO] Product is not loaded, retrying - %v", t.ProductSKU)
				// time.Sleep(1500 * time.Millisecond)
				continue
			case errTaskBanned:
				log.Printf("[WARN] Task is banned, retrying - %v", t.ProductSKU)
				t.SetProxy()
				challengeErr := t.GetCookies()
				if challengeErr == nil {
					log.Printf("[INFO] Set Distil Cookies - %v", t.ProductSKU)
					t.RequestCount = 1
				} else {
					log.Printf("[ERROR] Unhandled Error (Challenge) - %v - %v", challengeErr.Error(), t.ProductSKU)
				}

				time.Sleep(2500 * time.Millisecond)
				continue
			default:
				log.Printf("[ERROR] Unhandled Error - %v - %v", err.Error(), t.ProductSKU)
				t.SetProxy()
				challengeErr := t.GetCookies()
				if challengeErr == nil {
					log.Printf("[INFO] Set Distil Cookies - %v", t.ProductSKU)
					t.RequestCount = 1
				} else {
					log.Printf("[ERROR] Unhandled Error (Challenge) - %v - %v", challengeErr.Error(), t.ProductSKU)
				}
				time.Sleep(2500 * time.Millisecond)
				continue
			}
		}

		if len(sizeMap) == 0 {
			log.Printf("[INFO] Size map for product is empty, retrying - %v", t.ProductSKU)
			// time.Sleep(1500 * time.Millisecond)
			continue
		}

		log.Printf("[INFO] Gathered size map - %v", t.ProductSKU)
		t.CheckUpdate(sizeMap)
		// time.Sleep(1500 * time.Millisecond)
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
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		log.Printf("[INFO] Running Proxy (%v) - %v", proxyURL.String(), t.ProductSKU)
	} else {
		log.Printf("[WARN] Running Proxyless - %v", t.ProductSKU)
	}
}

func (t *endTask) GetChallengeLocation() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://distilnetworks.endservices.info", nil)

	if err != nil {
		return "", err
	}

	req.Host = "www.endclothing.com"
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.endclothing.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0")

	resp, err := t.Client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	html, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return "", err
	}

	if val, ok := html.Find(`script[src^="/ec"]`).Attr("src"); ok {
		return val, nil
	}

	return "", errChallengeNoPath
}

func (t *endTask) GetPayload() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://production.c9ext2p5vs.eu-west-2.elasticbeanstalk.com/generate", nil)

	if err != nil {
		return "", err
	}

	req.Header.Set("X-Distil-API-Key", "6d9be079-d581-421f-a584-960b64dd652d")

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		var payloadResponse endPayload
		err := json.NewDecoder(resp.Body).Decode(&payloadResponse)

		if err != nil {
			return "", err
		}

		if payloadResponse.Success {
			return payloadResponse.Payload, nil
		}
	}

	return "", errChallengeFailed
}

func (t *endTask) GetCookies() error {
	t.Client.Jar = nil
	challengePath, err := t.GetChallengeLocation()

	if err != nil {
		return err
	}

	payload, err := t.GetPayload()

	if err != nil {
		return err
	}

	form := url.Values{}

	form.Add("p", payload)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://distilnetworks.endservices.info%v", challengePath), strings.NewReader(form.Encode()))

	if err != nil {
		return err
	}

	req.Host = "www.endclothing.com"
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.endclothing.com/gb/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0")

	resp, err := t.Client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		cookies := resp.Cookies()

		if len(cookies) > 0 {
			jar, err := cookiejar.New(nil)

			if err != nil {
				return err
			}

			jar.SetCookies(siteURL, cookies)

			t.Client.Jar = jar
			return nil
		}
	}

	return errChallengeFailed
}

func (t *endTask) PurgeURL(productURL string) {
	if t.RequestCount%25 == 0 || t.RequestCount == 0 {
		t.SetProxy()
		err := t.GetCookies()
		if err != nil {
			log.Printf("[ERROR] Unhandled Error (Challenge) - %v - %v", err.Error(), t.ProductSKU)
		} else {
			log.Printf("[INFO] Set Distil Cookies - %v", t.ProductSKU)
		}
	}

	req, err := http.NewRequest("PURGE", productURL, nil)

	if err != nil {
		return
	}

	req.Host = "api2.endclothing.com"
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.endclothing.com/gb/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0")

	if t.Client.Jar != nil {
		cookies := t.Client.Jar.Cookies(siteURL)
		cookieArr := []string{}

		for _, cookie := range cookies {
			cookieArr = append(cookieArr, fmt.Sprintf("%v=%v;", cookie.Name, cookie.Value))
		}

		req.Header.Set("Cookie", strings.Join(cookieArr, ","))
	}

	resp, err := t.Client.Do(req)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	t.RequestCount++

	if resp.StatusCode == 200 {
		log.Printf("[INFO] Purged URL - %v - %v", t.ProductSKU, productURL)
	} else {
		log.Printf("[WARN] Failed to Purge URL - %v - %v", t.ProductSKU, productURL)
	}
}

func (t *endTask) GetSizes(productURL string) (map[string]bool, error) {
	if t.RequestCount%25 == 0 || t.RequestCount == 0 {
		t.SetProxy()
		err := t.GetCookies()
		if err != nil {
			log.Printf("[ERROR] Unhandled Error (Challenge) - %v - %v", err.Error(), t.ProductSKU)
		} else {
			log.Printf("[INFO] Set Distil Cookies - %v", t.ProductSKU)
		}
	}

	req, err := http.NewRequest(http.MethodGet, productURL, nil)

	if err != nil {
		return nil, err
	}

	req.Host = "api2.endclothing.com"
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.endclothing.com/gb/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0")

	if t.Client.Jar != nil {
		cookies := t.Client.Jar.Cookies(siteURL)
		cookieArr := []string{}

		for _, cookie := range cookies {
			cookieArr = append(cookieArr, fmt.Sprintf("%v=%v;", cookie.Name, cookie.Value))
		}

		req.Header.Set("Cookie", strings.Join(cookieArr, ","))
	}

	resp, err := t.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	t.RequestCount++

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

		if product.InStock && product.IsSalable {
			sizesAvailable := false
			sizeMap := make(map[string]bool)
			for _, sizeOption := range product.Options {
				if sizeOption.AttributeID == "173" && sizeOption.Label == "Size" {
					if len(sizeOption.Values) > 0 {
						sizesAvailable = true
						for _, individualSize := range sizeOption.Values {
							t.IndexMap[individualSize.Label] = individualSize.Index
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
	restock := &restockObject{
		SKU: t.ProductSKU,
	}
	updateAvailable := false
	for size, stockAvailable := range sizeMap {
		if sizeInstock, sizeExists := t.SizeMap[size]; sizeExists {
			if !sizeInstock && stockAvailable {
				restock.SizeArray = append(restock.SizeArray, t.IndexMap[size])
				updateAvailable = true
			}
		} else if stockAvailable {
			restock.SizeArray = append(restock.SizeArray, t.IndexMap[size])
			updateAvailable = true
		}
	}

	t.SizeMap = sizeMap

	if updateAvailable {
		if t.FirstRun {
			log.Printf("[INFO] Ignoring first run update - %v", t.ProductSKU)
		} else {
			log.Printf("[INFO] Update available - %v", t.ProductSKU)
			go t.SendRestock(restock)
			for _, webhookURL := range config.WebhookUrls {
				go t.SendUpdate(webhookURL)
			}
		}
	} else {
		log.Printf("[INFO] No update available - %v", t.ProductSKU)
	}
}

func (t *endTask) SendRestock(restock *restockObject) {
	restockPayload, err := json.Marshal(restock)

	if err != nil {
		log.Printf("[ERROR] [RESTOCK SERVER] %v - %v", t.ProductSKU, err.Error())
		return
	}

	req, err := http.NewRequest(http.MethodPost, config.RestockServer, bytes.NewBuffer(restockPayload))

	if err != nil {
		log.Printf("[ERROR] [RESTOCK SERVER] %v - %v", t.ProductSKU, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("[ERROR] [RESTOCK SERVER] %v - %v", t.ProductSKU, err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Printf("[SUCCESS] Restock sent - %v", t.ProductSKU)
	} else {
		log.Printf("[ERROR] [RESTOCK SERVER] Restock failed to send - %v - %v", resp.StatusCode, t.ProductSKU)
	}
}

func (t *endTask) SendUpdate(webhookURL string) {
	webhook := &discordWebhook{}

	webhookEmbed := discordEmbed{
		Title: t.ProductInfo.Name,
		URL:   fmt.Sprintf("%v?/%v=%v", t.ProductInfo.ProductURL, uniuri.NewLen(4), uniuri.NewLen(4)),
		Color: 1,
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
		Text:    fmt.Sprintf("assist by @afraidlabs | END • %v", time.Now().Format("15:04:05.000")),
		IconURL: "https://i.imgur.com/fOrEhkz.jpg",
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
