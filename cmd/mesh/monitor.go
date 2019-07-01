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

	"github.com/PuerkitoBio/goquery"
)

var (
	errInQueue    = errors.New("Task in queue")
	errNoWishlist = errors.New("No wishlist available")
)

func (t *meshFrontendTask) Monitor() {

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
		log.Printf("[WARN] Running Proxyless - %v - %v", t.SKU, t.SiteCode)
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

func (t *meshFrontendTask) GetSizes() (map[string]meshProductSKU, error) {

	return nil, nil
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

		log.Printf("[WARN] Item may have been added to wishlist, assuming failure - %v - %v", t.SKU, t.SiteCode)
		return nil, nil

	case 502:
		log.Printf("[WARN] Item could not be wishlisted - %v - %v", t.SKU, t.SiteCode)
		return nil, nil
	case 403:
		return nil, fmt.Errorf("Detected Ban (Frontend) - %v - %v", t.SKU, t.SiteCode)
	default:
		return nil, fmt.Errorf("Invalid Status Code (Frontend) - %v - %v", t.SKU, t.SiteCode)
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
			t.HandleQueue(req.URL.String())
			return "", errInQueue
		}

		wishlistID, wishlistExists := doc.Find(fmt.Sprintf(`*[data-sku="%v%v"]`, t.SKU, t.Site.SKUSuffix)).Attr("data-wishlistid")

		if wishlistExists {
			log.Printf("[INFO] Found Wishlist - %v - %v - %v", wishlistID, t.SKU, t.SiteCode)
			return wishlistID, nil
		}
		log.Printf("[WARN] No Wishlist - %v - %v", t.SKU, t.SiteCode)
		return "", nil
	default:
		return "", fmt.Errorf("Invalid Status Code (Frontend - GetWishlistID) - %v - %v - %v", resp.StatusCode, t.SKU, t.SiteCode)
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

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/wishlists/ajax/%v/", t.Site.SiteURL, t.WishlistID), nil)

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
		var wishlist meshFrontendWishlist
		err = json.NewDecoder(resp.Body).Decode(&wishlist)

		if err != nil {
			return nil, err
		}

		if wishlist.Content != nil {
			log.Printf("[INFO] Wishlist Not Empty - %v - %v - %v", t.WishlistID, t.SKU, t.SiteCode)
		} else {
			log.Printf("[WARN] Wishlist Empty - %v - %v - %v", t.WishlistID, t.SKU, t.SiteCode)
		}

		return &wishlist, nil
	default:
		return nil, fmt.Errorf("Invalid Status Code (Frontend - GetWishlist) - %v - %v - %v", resp.StatusCode, t.SKU, t.SiteCode)
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
		log.Printf("Error (Queue Bruter) - %v - %v", t.SKU, err.Error())
		t.SetProxy()
	}

	if queuePass != nil {
		t.SessionCookies[queuePassCookie] = queuePass
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
		return nil, fmt.Errorf("Detected Ban (Frontend - Queue Brute) - %v - %v", t.SKU, t.SiteCode)
	default:
		return nil, fmt.Errorf("Unknown Status Code (Frontend - Queue Brute) - %v - %v", t.SKU, t.SiteCode)
	}
}

func (t *meshBackendTask) Monitor() {

}
