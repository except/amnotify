package main

import (
	"net/http"
)

type meshSiteConfig map[string]*meshSite

type meshSite struct {
	SiteName    string   `json:"SiteName"`
	SiteURL     string   `json:"SiteUrl"`
	UserAgent   string   `json:"UserAgent"`
	StoreCode   string   `json:"StoreCode"`
	APIKey      string   `json:"APIKey"`
	HawkID      string   `json:"HawkID"`
	HawkSecret  string   `json:"HawkSecret"`
	SKUSuffix   string   `json:"SKUSuffix"`
	WebhookUrls []string `json:"WebhookUrls"`
}

type meshConfig struct {
	ProxyArray []string            `json:"ProxyArray"`
	Tasks      []meshConfigProduct `json:"Tasks"`
}

type meshConfigProduct struct {
	SKU   string   `json:"SKU"`
	Sites []string `json:"Sites"`
}

type meshFrontendTask struct {
	SKU            string
	WishlistID     string
	Site           *meshSite
	SiteCode       string
	Client         *http.Client
	ProductInfo    *meshProductInfo
	SessionCookies map[string]*http.Cookie
	ProductSKUMap  map[string]meshProductSKU
}

type meshBackendTask struct {
	SKU           string
	Site          *meshSite
	SiteCode      string
	Client        *http.Client
	ProductInfo   *meshProductInfo
	ProductSKUMap map[string]meshProductSKU
}

type meshProductInfo struct {
	Name, Price, ImageURL string
}

type meshFrontendWishlist struct {
	Content []struct {
		Products []struct {
			Product struct {
				ID        string `json:"ID"`
				SKU       string `json:"SKU"`
				Name      string `json:"name"`
				MainImage string `json:"mainImage"`
				Price     struct {
					Amount   string `json:"amount"`
					Currency string `json:"currency"`
				} `json:"price"`
				Options map[string]meshProductSKU `json:"options"`
			} `json:"product"`
		} `json:"products"`
	} `json:"content"`
}

type meshBackendProduct struct {
	ID        string `json:"ID"`
	SKU       string `json:"SKU"`
	Name      string `json:"name"`
	MainImage string `json:"mainImage"`
	Price     struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"price"`
	Options map[string]meshProductSKU `json:"options"`
}

type meshProductSKU struct {
	SKU         string `json:"SKU"`
	Size        string `json:"size"`
	StockStatus string `json:"stockStatus"`
}

type meshWishlistPayload struct {
	Label       interface{} `json:"label"`
	IsPublic    bool        `json:"isPublic"`
	ProductSkus []string    `json:"productSkus"`
}

type meshWishlistMessage struct {
	Message string `json:"message"`
}

type discordWebhook struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title     string                `json:"title"`
	URL       string                `json:"url"`
	Color     int                   `json:"color"`
	Footer    discordEmbedFooter    `json:"footer"`
	Thumbnail discordEmbedThumbnail `json:"thumbnail"`
	Fields    []discordEmbedField   `json:"fields"`
}

type discordEmbedFooter struct {
	IconURL string `json:"icon_url"`
	Text    string `json:"text"`
}

type discordEmbedThumbnail struct {
	URL string `json:"url"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}
