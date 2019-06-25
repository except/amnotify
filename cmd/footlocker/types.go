package main

import (
	"sync"
)

type ftlConfig struct {
	SKUArray   []ftlSKU              `json:"SKUArray"`
	ProxyArray []string              `json:"ProxyArray"`
	Regions    map[string]*ftlRegion `json:"Regions"`
}

type ftlSKU struct {
	SKU     string   `json:"SKU"`
	Regions []string `json:"Regions"`
}

type ftlRegion struct {
	BaseURL     string   `json:"BaseUrl"`
	WebhookUrls []string `json:"WebhookUrls"`
}

type ftlContent struct {
	Content string `json:"content"`
}

type ftlTask struct {
	SKU         string
	FirstRun    bool
	PageRemoved bool

	Region      *ftlRegion
	RegionName  string
	ProductInfo *ftlProdInfo

	sync.Mutex
	Inventory map[string]ftlSize
}

type ftlProdInfo struct {
	Name, Price, URL string
}

type ftlSize struct {
	InventoryLevel  string    `json:"inventoryLevel"`
	QuantityWarning string    `json:"quantityWarning"`
	SizeValue       string    `json:"sizeValue"`
	QuantityMessage string    `json:"quantityMessage"`
	QuantityOptions []float64 `json:"quantityOptions"`
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
