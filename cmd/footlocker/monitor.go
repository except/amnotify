package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/dchest/uniuri"

	"github.com/PuerkitoBio/goquery"
)

func (p *ftlTask) beginMonitor() {
	p.setProxy()
	for {
		productInventory, err := p.getInventory()

		if err != nil {
			log.Printf("Error %v - %v", p.SKU, err.Error())
			p.setProxy()
			time.Sleep(1500 * time.Millisecond)
			continue

		}

		if productInventory == nil {
			if p.PageRemoved {
				log.Printf("[INFO] Page Removed - %v - %v", p.SKU, p.RegionName)
			} else {
				log.Printf("[INFO] No Sizes Available - %v - %v", p.SKU, p.RegionName)
			}

			time.Sleep(750 * time.Millisecond)
			continue
		}

		p.checkUpdate(productInventory)

		time.Sleep(750 * time.Millisecond)
	}
}

func (p *ftlTask) setProxy() {
	if len(config.ProxyArray) > 0 {
		proxy := config.ProxyArray[rand.Intn(len(config.ProxyArray))]

		proxyURL, err := url.Parse(proxy)

		if err != nil {
			log.Printf("Error %v - %v", p.SKU, err.Error())
			log.Printf("[WARN] Running Proxyless - %v - %v", p.SKU, p.RegionName)
			return
		}

		p.Client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		log.Printf("[INFO] Running Proxy (%v) - %v - %v", proxyURL.String(), p.SKU, p.RegionName)
	} else {
		log.Printf("[WARN] Running Proxyless - %v - %v", p.SKU, p.RegionName)
	}
}

func (p *ftlTask) getInventory() (map[string]ftlSize, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/ViewProductTile-ProductVariationSelect?BaseSKU=%v&InventoryServerity=StandardCatalog&%v=%v", p.Region.BaseURL, p.SKU, uniuri.NewLen(8), uniuri.NewLen(8)), nil)

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")

	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {

		if p.ProductInfo == nil {
			productInfo, err := p.pullProdInfo()
			if err != nil {
				log.Println(err.Error())
			} else {
				p.ProductInfo = productInfo
			}
		}

		p.PageRemoved = false

		var content ftlContent

		err = json.NewDecoder(resp.Body).Decode(&content)

		if err != nil {
			return nil, err
		}

		document, err := goquery.NewDocumentFromReader(strings.NewReader(content.Content))

		if err != nil {
			return nil, err
		}

		var ftlProdMap map[string]ftlSize

		ftlProductJSON, _ := document.Find(fmt.Sprintf("div[data-product-variation-info=\"%v\"]", p.SKU)).Attr("data-product-variation-info-json")
		err = json.Unmarshal([]byte(ftlProductJSON), &ftlProdMap)

		if err != nil {
			return nil, err
		}

		return ftlProdMap, nil

	} else if resp.StatusCode == 302 {
		if p.FirstRun {
			p.FirstRun = false
		}

		p.PageRemoved = true
		return nil, nil
	}

	return nil, fmt.Errorf("[WARN] Invalid Status Code (Product Inventory) - %v - %v", resp.StatusCode, p.SKU)
}

func (p *ftlTask) pullProdInfo() (*ftlProdInfo, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/ViewProductTile-ProductTileBasicJSON?BaseSKU=%v", p.Region.BaseURL, p.SKU), nil)

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")

	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var content ftlContent

		err = json.NewDecoder(resp.Body).Decode(&content)

		if err != nil {
			return nil, err
		}

		document, err := goquery.NewDocumentFromReader(strings.NewReader(content.Content))

		if err != nil {
			return nil, err
		}

		productName := document.Find("span[itemprop=\"name\"]").Text()
		productPrice := document.Find("a > div > span > span").Text()
		productURL, _ := document.Find("a").Attr("href")

		productInfo := &ftlProdInfo{
			Name:  productName,
			Price: productPrice,
			URL:   productURL,
		}

		return productInfo, nil

	} else if resp.StatusCode == 302 {

		return nil, fmt.Errorf("[INFO] Page Info Redirecting - %v", p.SKU)
	}

	return nil, fmt.Errorf("[WARN] Invalid Status Code (Page Info) - %v - %v", resp.StatusCode, p.SKU)
}

func (p *ftlTask) checkUpdate(productInventory map[string]ftlSize) {
	updateAvailable := false

	for ftlSizeSKU, ftlSKUStatus := range productInventory {
		ftlPrevSKUStatus, ftlSKUAvailable := p.Inventory[ftlSizeSKU]

		if ftlSKUAvailable {
			if ftlPrevSKUStatus.InventoryLevel == "RED" {
				if ftlSKUStatus.InventoryLevel != "RED" {
					updateAvailable = true
				}
			}

			p.Inventory[ftlSizeSKU] = ftlSKUStatus

		} else {
			if ftlSKUStatus.InventoryLevel != "RED" {
				updateAvailable = true
			}

			p.Inventory[ftlSizeSKU] = ftlSKUStatus
		}
	}

	if updateAvailable {
		log.Printf("[INFO] Product Update Detected - %v - %v", p.SKU, p.RegionName)

		if !p.FirstRun {
			for _, webhookURL := range p.Region.WebhookUrls {
				go p.notifyWebhook(webhookURL)
			}
		} else {
			log.Printf("[INFO] Ignoring Product Update - %v - %v", p.SKU, p.RegionName)
			p.FirstRun = false
		}
	} else {
		log.Printf("[INFO] No Restock Detected - %v - %v", p.SKU, p.RegionName)
	}
}

func (p *ftlTask) notifyWebhook(webhookURL string) {
	hookStruct := &discordWebhook{}

	hookEmbed := discordEmbed{
		Color: 16721733,
	}

	priceField := discordEmbedField{
		Name:   "Price",
		Inline: true,
	}

	if p.ProductInfo != nil {
		hookEmbed.Title = p.ProductInfo.Name
		hookEmbed.URL = p.ProductInfo.URL
		priceField.Value = p.ProductInfo.Price
	} else {
		hookEmbed.Title = p.SKU
		hookEmbed.URL = p.Region.BaseURL
		priceField.Value = "N/A"
	}

	if hookEmbed.Title == "" {
		hookEmbed.Title = p.SKU
	}

	if hookEmbed.URL == "" {
		hookEmbed.URL = p.Region.BaseURL
	}

	if priceField.Value == "" {
		priceField.Value = "N/A"
	}

	hookEmbed.Fields = append(hookEmbed.Fields, priceField)

	hookEmbed.Thumbnail = discordEmbedThumbnail{
		URL: fmt.Sprintf("https://runnerspoint.scene7.com/is/image/rpe/%v_01?wid=512", p.SKU),
	}

	hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
		Name:   "Product SKU",
		Value:  p.SKU,
		Inline: true,
	})

	hookEmbed.Footer = discordEmbedFooter{
		Text:    fmt.Sprintf("AMNotify | Footlocker %v • %v", p.RegionName, time.Now().Format("15:04:05.000")),
		IconURL: "https://i.imgur.com/vv2dyGR.png",
	}

	var availableSKUs []string

	for ftlSizeSKU, ftlSKUStatus := range p.Inventory {
		if ftlSKUStatus.InventoryLevel != "RED" {
			availableSKUs = append(availableSKUs, ftlSizeSKU)
		}
	}

	sort.Strings(availableSKUs)

	var availableSizeString []string

	for _, ftlSKU := range availableSKUs {
		var sizePrefix string

		switch p.RegionName {
		case "GB":
			sizePrefix = "UK"
		default:
			sizePrefix = "EU"
		}

		availableSizeString = append(availableSizeString, fmt.Sprintf("[%v %v](http://amnotify.io/ftl.html?SKU=%v&region=%v)", sizePrefix, p.Inventory[ftlSKU].SizeValue, ftlSKU, p.RegionName))
	}

	if len(availableSizeString) > 0 {
		hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
			Name:   "Size Availability",
			Value:  strings.Join(availableSizeString, "\n"),
			Inline: true,
		})
	}

	if len(availableSKUs) > 0 {
		hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
			Name:   "SKU Availability",
			Value:  strings.Join(availableSKUs, "\n"),
			Inline: true,
		})
	}

	hookStruct.Embeds = append(hookStruct.Embeds, hookEmbed)

	webhookPayload, err := json.Marshal(hookStruct)

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", p.SKU, err.Error())
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(webhookPayload))

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", p.SKU, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", p.SKU, err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		log.Printf("[SUCCESS] Webhook Sent - %v - %v", p.SKU, p.RegionName)
	} else if resp.StatusCode == 429 {
		log.Printf("[WARN] Ratelimited - %v", p.SKU)
		time.Sleep(5 * time.Second)
		p.notifyWebhook(webhookURL)
	} else {
		log.Printf("[WARN] Invalid Status - %v - %v", p.SKU, resp.Status)
	}

	return
}
