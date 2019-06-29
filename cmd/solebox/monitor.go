package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/dchest/uniuri"

	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func (p *sbxProduct) launchMonitor() {
	for {
		sizes, err := p.getSizes()

		if err != nil {
			log.Printf("[ERROR] %v - %v", p.URL, err.Error())
			continue
		}

		if sizes != nil {
			p.checkUpdate(sizes)
		} else {
			if p.PageRemoved {
				log.Printf("[INFO] No Stock Update (Page Removed) - %v", p.URL)
			} else {
				log.Printf("[INFO] No Stock Update (No Sizes) - %v", p.ProductInfo.ProductName)
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func (p *sbxProduct) getSizes() ([]*sbxSize, error) {
	productURL := strings.Replace(p.URL, "www.solebox.com", "cdn.solebox.com", 1)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v?%v=%v", productURL, uniuri.NewLen(8), uniuri.NewLen(8)), nil)

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
		p.PageRemoved = false

		page, err := goquery.NewDocumentFromReader(resp.Body)

		if err != nil {
			return nil, err
		}

		if p.ProductInfo == nil {
			productName, _ := page.Find(`meta[itemprop="name"]`).Attr("content")
			productPrice, _ := page.Find(`meta[itemprop="price"]`).Attr("content")
			productImage, _ := page.Find(`#zoom1`).Attr("href")

			p.ProductInfo = &sbxProductInfo{
				ProductName:  productName,
				ProductPrice: productPrice,
				ProductImage: productImage,
			}
		}

		var availableSizes []*sbxSize

		page.Find(".size").Each(func(index int, size *goquery.Selection) {
			sizeAvailable := true
			if size.HasClass("inactive") {
				sizeAvailable = false
			}
			sizeNode := size.Find(".selectSize")
			sizeName, _ := sizeNode.Attr("data-size-us")
			sizeID, _ := sizeNode.Attr("id")

			sizeStruct := &sbxSize{
				SizeName:  sizeName,
				SizeAID:   sizeID,
				Available: sizeAvailable,
			}

			p.setSize(sizeStruct)

			availableSizes = append(availableSizes, sizeStruct)
		})

		return availableSizes, nil

	} else if resp.StatusCode == 302 || resp.StatusCode == 404 {
		// OOS
		// Disable First Run
		if p.FirstRun {
			p.FirstRun = false
		}

		p.PageRemoved = true

		return nil, nil
	}

	return nil, nil
}

func (p *sbxProduct) setAvailable(size *sbxSize) {
	p.Lock()
	p.SizeAvailability[size.SizeAID] = true
	p.Unlock()
}

func (p *sbxProduct) setUnavailable(size *sbxSize) {
	p.Lock()
	p.SizeAvailability[size.SizeAID] = false
	p.Unlock()
}

func (p *sbxProduct) setSize(size *sbxSize) {
	p.Lock()
	p.SizeMap[size.SizeAID] = size.SizeName
	p.Unlock()
}

func (p *sbxProduct) checkUpdate(sizes []*sbxSize) {
	stockUpdateExists := false

	for _, size := range sizes {
		sizeAvailability, sizeExists := p.SizeAvailability[size.SizeAID]
		if sizeExists && sizeAvailability != size.Available {
			switch size.Available {
			case true:
				//Update Available
				log.Printf("[INFO] Size Update Instock - %v - %v", size.SizeName, p.ProductInfo.ProductName)
				p.setAvailable(size)
				stockUpdateExists = true
			case false:
				//Update Unavailable
				p.setUnavailable(size)
			}
		} else if !sizeExists {
			switch size.Available {
			case true:
				// Now Available
				log.Printf("[INFO] Size Now Instock - %v - %v", size.SizeName, p.ProductInfo.ProductName)
				p.setAvailable(size)
				stockUpdateExists = true
			case false:
				// Now Unavailable
				p.setUnavailable(size)
			}
		} else {

		}

	}

	if stockUpdateExists && !p.FirstRun {
		for _, webhookURL := range config.WebhookUrls {
			go p.sendUpdate(webhookURL)
		}

	} else if p.FirstRun {
		p.FirstRun = false
	} else if !stockUpdateExists {
		log.Printf("[INFO] No Stock Update - %v", p.ProductInfo.ProductName)
	}
}

func (p *sbxProduct) sendUpdate(webhookURL string) {
	hookStruct := &discordWebhook{}

	hookEmbed := discordEmbed{
		Title: p.ProductInfo.ProductName,
		URL:   p.URL,
		Color: 16721733,
	}

	hookEmbed.Thumbnail = discordEmbedThumbnail{
		URL: p.ProductInfo.ProductImage,
	}

	hookEmbed.Footer = discordEmbedFooter{
		Text:    fmt.Sprintf("AMNotify | Solebox • %v", time.Now().Format("15:04:05.000")),
		IconURL: "https://i.imgur.com/vv2dyGR.png",
	}

	hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
		Name:   "Price",
		Value:  fmt.Sprintf("€%v", p.ProductInfo.ProductPrice),
		Inline: true,
	})

	hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
		Name:   "Important Links",
		Value:  "[Start Card Checkout](https://www.solebox.com/index.php?cl=payment#payment_gs_kk_saferpay)\n[Start PayPal Checkout](https://www.solebox.com/index.php?pp=redirect&cl=payment&fnc=validatepayment&paymentid=globalpaypal)",
		Inline: true,
	})

	var availableSizeArr []string
	var unavailableSizeArr []string

	for sizeAID := range p.SizeAvailability {
		if p.SizeAvailability[sizeAID] {
			availableSizeArr = append(availableSizeArr, sizeAID)
		} else {
			unavailableSizeArr = append(unavailableSizeArr, sizeAID)
		}
	}

	sort.Strings(availableSizeArr)
	sort.Strings(unavailableSizeArr)

	var availableSizeStringArr []string
	var unavailableSizeStringArr []string

	for _, sizeAID := range availableSizeArr {
		sizeName := p.SizeMap[sizeAID]
		sizeString := fmt.Sprintf("[US %v](https://www.solebox.com/index.php?fnc=changebasket&aproducts[0][aid]=%v&aproducts[0][am]=1&cl=basket&lang=1)", sizeName, sizeAID)
		availableSizeStringArr = append(availableSizeStringArr, sizeString)
	}

	for _, sizeAID := range unavailableSizeArr {
		sizeName := p.SizeMap[sizeAID]
		sizeString := fmt.Sprintf("[~~US %v~~](https://www.solebox.com/index.php?fnc=changebasket&aproducts[0][aid]=%v&aproducts[0][am]=1&cl=basket&lang=1)", sizeName, sizeAID)
		unavailableSizeStringArr = append(unavailableSizeStringArr, sizeString)
	}

	if len(availableSizeStringArr) > 0 {
		hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
			Name:   "Size Availability",
			Value:  strings.Join(availableSizeStringArr, "\n"),
			Inline: true,
		})
	}

	if len(unavailableSizeStringArr) > 0 {
		hookEmbed.Fields = append(hookEmbed.Fields, discordEmbedField{
			Name:   "Unavailable Sizes",
			Value:  strings.Join(unavailableSizeStringArr, "\n"),
			Inline: true,
		})

	}

	hookStruct.Embeds = append(hookStruct.Embeds, hookEmbed)

	webhookPayload, err := json.Marshal(hookStruct)

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", p.ProductInfo.ProductName, err.Error())
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(webhookPayload))

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", p.ProductInfo.ProductName, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("[ERROR] [WEBHOOK] %v - %v", p.ProductInfo.ProductName, err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		log.Printf("[SUCCESS] Webhook Sent - %v", p.ProductInfo.ProductName)
	} else if resp.StatusCode == 429 {
		log.Printf("[WARN] Ratelimited - %v", p.ProductInfo.ProductName)
		time.Sleep(5 * time.Second)
		p.sendUpdate(webhookURL)
	} else {
		log.Printf("[WARN] Invalid Status - %v - %v", p.ProductInfo.ProductName, resp.Status)
	}

	return
}
