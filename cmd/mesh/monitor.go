package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/dchest/uniuri"

	"github.com/PuerkitoBio/goquery"
)

var (
	errInQueue    = errors.New("Task in queue")
	errNoWishlist = errors.New("No wishlist available")
	errItemOOS    = errors.New("Item is OOS")
	errTaskBanned = errors.New("403 Detected")
)

func (t *meshFrontendTask) Monitor() {
	log.Printf("[INFO] Start Tasking (Frontend) - %v - %v", t.SKU, t.SiteCode)
	t.SetProxy()

	for {
		SKUMap, err := t.GetSizes()

		if err != nil {
			log.Printf("Unhandled error (Frontend) - %v", err.Error())
			log.Printf("[INFO] Resetting task (Frontend) - %v - %v", t.SKU, t.SiteCode)
			t.ResetTask()
			continue
		}

		t.CheckUpdate(SKUMap)

		time.Sleep(3 * time.Second)
	}
}

func (t *meshFrontendTask) SetProxy() {
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

		log.Printf("[INFO] Running Proxy (%v) (Frontend) - %v - %v", proxyURL.String(), t.SKU, t.SiteCode)
	} else {
		log.Printf("[WARN] Running Proxyless (Frontend) - %v - %v", t.SKU, t.SiteCode)
	}
}

func (t *meshFrontendTask) SetCookies() error {
	jar, err := cookiejar.New(nil)

	if err != nil {
		return err
	}

	var cookies []*http.Cookie

	for _, cookie := range t.SessionCookies {
		cookies = append(cookies, cookie)
	}

	siteURL, err := url.Parse(t.Site.SiteURL)

	if err != nil {
		return err
	}

	jar.SetCookies(siteURL, cookies)

	t.Client.Jar = jar

	return nil
}

func (t *meshFrontendTask) ResetTask() {
	t.Client = &http.Client{
		Timeout: 15 * time.Second,
	}

	t.WishlistID = ""

	t.SessionCookies = make(map[string]*http.Cookie)
	t.ProductSKUMap = make(map[string]meshProductSKU)

	t.SetProxy()
}

func (t *meshFrontendTask) GetSizes() (map[string]meshProductSKU, error) {
	var err error

	if t.WishlistID == "" {
		var session *http.Cookie

		for session == nil || err != nil {
			session, err = t.AddToWishlist()

			if err != nil {
				switch err {
				case errInQueue:
					log.Printf("[INFO] Retrying (Frontend - AddToWishlist) - %v - %v", t.SKU, t.SiteCode)
					continue
				case errItemOOS:
					log.Printf("[INFO] Delaying retry for OOS item (Frontend - AddToWishlist) - %v - %v", t.SKU, t.SiteCode)
					time.Sleep(1 * time.Second)
					continue
				case errTaskBanned:
					t.SetProxy()
					log.Printf("[INFO] Delaying retry for banned task (Frontend - AddToWishlist) - %v - %v", t.SKU, t.SiteCode)
					time.Sleep(1 * time.Second)
					continue
				default:
					return nil, err
				}

			}

			if session != nil && err == nil {
				t.SessionCookies[sessionCookie] = session
				log.Printf("[INFO] Set session cookie - %v - %v", t.SKU, t.SiteCode)
				break
			}
		}

		var wishlistID string

		for wishlistID == "" || err != nil {
			wishlistID, err = t.GetWishlistID()

			if err != nil {
				switch err {
				case errInQueue:
					log.Printf("[INFO] Retrying (Frontend - GetWishlistID) - %v - %v", t.SKU, t.SiteCode)
					continue
				case errItemOOS:
					log.Printf("[INFO] Delaying retry for OOS item (Frontend - GetWishlistID) - %v - %v", t.SKU, t.SiteCode)
					time.Sleep(1 * time.Second)
					continue
				case errTaskBanned:
					t.SetProxy()
					log.Printf("[INFO] Delaying retry for banned task (Frontend - GetWishlistID) - %v - %v", t.SKU, t.SiteCode)
					time.Sleep(1 * time.Second)
					continue
				case errNoWishlist:
					log.Printf("[INFO] Resetting task as no wishlist detected (Frontend - GetWishlistID) - %v - %v", t.SKU, t.SiteCode)
					t.ResetTask()
					return t.GetSizes()
				default:
					return nil, err
				}
			}

			if wishlistID != "" && err == nil {
				t.WishlistID = wishlistID
				log.Printf("[INFO] Set WishlistID (Frontend)- %v - %v", t.SKU, t.SiteCode)
				break
			}
		}
	}

	var wishlist *meshFrontendWishlist

	for wishlist == nil || err != nil {
		wishlist, err = t.GetWishlist()

		if err != nil {
			switch err {
			case errInQueue:
				log.Printf("[INFO] Retrying (Frontend - GetWishlist) - %v - %v", t.SKU, t.SiteCode)
				continue
			case errItemOOS:
				log.Printf("[INFO] Delaying retry for OOS item (Frontend - GetWishlist) - %v - %v", t.SKU, t.SiteCode)
				time.Sleep(1 * time.Second)
				continue
			case errTaskBanned:
				t.SetProxy()
				log.Printf("[INFO] Delaying retry for banned task (Frontend - GetWishlist) - %v - %v", t.SKU, t.SiteCode)
				time.Sleep(1 * time.Second)
				continue
			case errNoWishlist:
				log.Printf("[INFO] Resetting task as no wishlist detected (Frontend - GetWishlist) - %v - %v", t.SKU, t.SiteCode)
				t.ResetTask()
				return t.GetSizes()
			default:
				return nil, err
			}
		}

		if wishlist != nil && err == nil {
			log.Printf("[INFO] Got wishlist response (Frontend) - %v - %v", t.SKU, t.SiteCode)
			break
		}
	}

	var SKUMap map[string]meshProductSKU

	for _, content := range wishlist.Content {
		for _, product := range content.Products {
			if product.Product.SKU == fmt.Sprintf("%v%v", t.SKU, t.Site.SKUSuffix) {

				if t.ProductInfo == nil {
					t.ProductInfo = &meshProductInfo{
						Name:     product.Product.Name,
						Price:    fmt.Sprintf("%v %v", product.Product.Price.Amount, product.Product.Price.Currency),
						ImageURL: product.Product.MainImage,
					}
				}

				SKUMap = product.Product.Options
			}
		}
	}

	return SKUMap, nil
}

func (t *meshFrontendTask) AddToWishlist() (*http.Cookie, error) {
	err := t.SetCookies()

	if err != nil {
		return nil, err
	}

	wishlistPayload := &meshWishlistPayload{
		Label:       nil,
		IsPublic:    false,
		ProductSkus: []string{fmt.Sprintf("%v%v", t.SKU, t.Site.SKUSuffix)},
	}

	wishlistBytes, err := json.Marshal(wishlistPayload)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v/wishlists/ajax", t.Site.SiteURL), bytes.NewBuffer(wishlistBytes))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := t.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		if t.DetectQueue(resp.Cookies()) {
			log.Printf("[WARN] Detected queue (Frontend - AddToWishlist) - %v - %v", t.SKU, t.SiteCode)
			t.HandleQueue(req.URL.String())
			return nil, errInQueue
		}

		var response meshWishlistMessage

		err = json.NewDecoder(resp.Body).Decode(&response)

		if err != nil {
			return nil, err
		}

		if response.Message == "Wishlist updated successfully" {
			log.Printf("[INFO] Item added to wishlist - %v - %v", t.SKU, t.SiteCode)
			for _, cookie := range resp.Cookies() {
				if cookie.Name == sessionCookie {
					log.Printf("[INFO] Found session cookie - %v - %v", t.SKU, t.SiteCode)
					return cookie, nil
				}
			}
		}

		log.Printf("Item may have been added to wishlist, assuming failure (Frontend) - %v - %v", t.SKU, t.SiteCode)
		return nil, errNoWishlist
	case 502:
		log.Printf("[WARN] Item could not be wishlisted (Frontend) - %v - %v", t.SKU, t.SiteCode)
		return nil, errItemOOS
	case 403:
		return nil, errTaskBanned
	default:
		return nil, fmt.Errorf("Invalid status code (Frontend) - %v - %v", t.SKU, t.SiteCode)
	}
}

func (t *meshFrontendTask) GetWishlistID() (string, error) {
	err := t.SetCookies()

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/wishlists/", t.Site.SiteURL), nil)

	if err != nil {
		return "", err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := t.Client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 200:
		if t.DetectQueue(resp.Cookies()) {
			log.Printf("[WARN] Detected queue (Frontend - GetWishlistID) - %v - %v", t.SKU, t.SiteCode)
			t.HandleQueue(req.URL.String())
			return "", errInQueue
		}

		wishlistID, wishlistExists := doc.Find(fmt.Sprintf(`*[data-sku="%v%v"]`, t.SKU, t.Site.SKUSuffix)).Attr("data-wishlistid")

		if wishlistExists {
			log.Printf("[INFO] Found Wishlist (Frontend) - %v - %v - %v", wishlistID, t.SKU, t.SiteCode)
			return wishlistID, nil
		}
		log.Printf("[WARN] No Wishlist (Frontend) - %v - %v", t.SKU, t.SiteCode)
		return "", nil
	case 403:
		return "", errTaskBanned
	default:
		return "", fmt.Errorf("Invalid status code (Frontend - GetWishlistID) - %v - %v - %v", resp.StatusCode, t.SKU, t.SiteCode)
	}
}

func (t *meshFrontendTask) GetWishlist() (*meshFrontendWishlist, error) {
	err := t.SetCookies()

	if err != nil {
		return nil, err
	}

	if t.WishlistID == "" {
		return nil, errNoWishlist
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/wishlists/ajax/%v/?%v=%v", t.Site.SiteURL, t.WishlistID, uniuri.NewLen(8), uniuri.NewLen(8)), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := t.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		if t.DetectQueue(resp.Cookies()) {
			log.Printf("[WARN] Detected queue (Frontend - GetWishlist) - %v - %v", t.SKU, t.SiteCode)
			t.HandleQueue(req.URL.String())
			return nil, errInQueue
		}

		var wishlist meshFrontendWishlist
		err = json.NewDecoder(resp.Body).Decode(&wishlist)

		if err != nil {
			return nil, err
		}

		if wishlist.Content != nil {
			log.Printf("[INFO] Wishlist not empty (Frontend) - %v - %v - %v", t.WishlistID, t.SKU, t.SiteCode)
		} else {
			log.Printf("[WARN] Wishlist empty (Frontend) - %v - %v - %v", t.WishlistID, t.SKU, t.SiteCode)
		}

		return &wishlist, nil
	case 403:
		return nil, errTaskBanned
	default:
		return nil, fmt.Errorf("Invalid status code (Frontend - GetWishlist) - %v - %v - %v", resp.StatusCode, t.SKU, t.SiteCode)
	}
}

func (t *meshFrontendTask) DetectQueue(cookies []*http.Cookie) bool {
	for _, cookie := range cookies {
		if cookie.Name == queueCookie {
			return true
		}
	}

	return false
}

func (t *meshFrontendTask) HandleQueue(queueURL string) {
	queuePass, err := t.QueueBrute(queueURL)

	if err != nil {
		log.Printf("Error (Frontend - Queue Bruter) - %v - %v", t.SKU, err.Error())
		t.SetProxy()
	}

	if queuePass != nil {
		log.Printf("[INFO] Passed queue (Frontend) - %v - %v - %v", queuePass.Value, t.SKU, t.SiteCode)
		t.SessionCookies[queuePassCookie] = queuePass
		return
	}

	t.HandleQueue(queueURL)
}

func (t *meshFrontendTask) QueueBrute(queueURL string) (*http.Cookie, error) {
	bruteClient := new(http.Client)

	bruteClient.Timeout = 15 * time.Second

	bruteClient.Transport = t.Client.Transport

	req, err := http.NewRequest(http.MethodHead, queueURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := bruteClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		if !t.DetectQueue(resp.Cookies()) {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == queuePassCookie {
					return cookie, nil
				}
			}
		}
		return nil, nil
	case 403:
		return nil, errTaskBanned
	default:
		return nil, fmt.Errorf("Invalid status code (Frontend - Queue Brute) - %v - %v", t.SKU, t.SiteCode)
	}
}

func (t *meshFrontendTask) CheckUpdate(SKUMap map[string]meshProductSKU) {
	updateAvailable := false

	for sizeName, productSKU := range SKUMap {
		if currentProductSKU, SKUExists := t.ProductSKUMap[sizeName]; SKUExists {
			if productSKU.StockStatus == "IN STOCK" && currentProductSKU.StockStatus == "OUT OF STOCK" {
				updateAvailable = true
			}
		} else {
			if productSKU.StockStatus == "IN STOCK" {
				updateAvailable = true
			}
		}

		t.ProductSKUMap[sizeName] = productSKU
	}

	if updateAvailable {
		log.Printf("[INFO] Product stock update detected (Frontend) - %v - %v", t.SKU, t.SiteCode)
	} else {
		log.Printf("[INFO] No product stock update (Frontend) - %v - %v", t.SKU, t.SiteCode)
	}

}

func (t *meshBackendTask) Monitor() {

}
