package main

import "net/http"

import "time"

import "sync"

type endConfig struct {
	ProductSKUs   []string `json:"ProductSKUs"`
	Proxies       []string `json:"Proxies"`
	WebhookUrls   []string `json:"WebhookUrls"`
	RestockServer string   `json:"RestockServer"`
}

type endCookies struct {
	mu  sync.Mutex
	Map map[string]time.Time
}

type endPayload struct {
	Success bool   `json:"success"`
	Payload string `json:"payload"`
}

type endProdInfo struct {
	Name, ProductURL, Price, ImageURL string
}

type endTask struct {
	Cookies    string
	ProductSKU string

	FirstRun       bool
	RequestCount   int
	PrevAvgLatency int64

	Client      *http.Client
	ProductInfo *endProdInfo

	LatencyArray []int64

	SizeMap  map[string]bool
	IndexMap map[string]string
}

type endProduct struct {
	ID                  int    `json:"id"`
	Sku                 string `json:"sku"`
	Name                string `json:"name"`
	Link                string `json:"link"`
	InStock             bool   `json:"in_stock"`
	IsSalable           bool   `json:"is_salable"`
	Price               int    `json:"price"`
	MediaGalleryEntries []struct {
		File string `json:"file"`
	} `json:"media_gallery_entries"`
	Options []struct {
		AttributeID string `json:"attribute_id"`
		Label       string `json:"label"`
		Values      []struct {
			Index   string `json:"index"`
			Label   string `json:"label"`
			InStock bool   `json:"in_stock"`
		} `json:"values"`
	} `json:"options"`
}

type restockObject struct {
	SKU       string   `json:"SKU"`
	SizeArray []string `json:"sizeArray"`
}

type discordWebhook struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title     string                `json:"title,omitempty"`
	URL       string                `json:"url,omitempty"`
	Color     int                   `json:"color,omitempty"`
	Footer    discordEmbedFooter    `json:"footer,omitempty"`
	Thumbnail discordEmbedThumbnail `json:"thumbnail,omitempty"`
	Fields    []discordEmbedField   `json:"fields,omitempty"`
}

type discordEmbedFooter struct {
	IconURL string `json:"icon_url,omitempty"`
	Text    string `json:"text,omitempty"`
}

type discordEmbedThumbnail struct {
	URL string `json:"url,omitempty"`
}

type discordEmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}
