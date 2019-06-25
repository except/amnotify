package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dchest/uniuri"

	"github.com/PuerkitoBio/goquery"
)

func (p *ftlTask) beginMonitor() {
	for {
		productInventory, err := p.getInventory()

		if err != nil {
			log.Printf("Error %v - %v", p.SKU, err.Error())
			time.Sleep(100 * time.Second)
			continue

		}

		if productInventory == nil {
			if p.PageRemoved {
				log.Printf("[INFO] Page Removed - %v - %v", p.SKU, p.RegionName)
			} else {
				log.Printf("[INFO] No Sizes Available - %v - %v", p.SKU, p.RegionName)
			}

			time.Sleep(100 * time.Second)
			continue
		}

		time.Sleep(100 * time.Second)
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

	if p.ProductInfo == nil {
		productInfo, err := p.pullProdInfo()
		if err != nil {
			log.Println(err.Error())
		} else {
			p.ProductInfo = productInfo
		}
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
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
	}

	return nil, nil
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

	resp, err := client.Do(req)

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
		ftlPrevSKUStatus, ftlSKUAvailable := productInventory[ftlSizeSKU]

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

		for _, webhookURL := range p.Region.WebhookUrls {
			go p.notifyWebhook(webhookURL)
		}
	}
}

func (p *ftlTask) notifyWebhook(webhookURL string) {

}
