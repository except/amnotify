package main

import (
	"net/http"
	"sync"
)

type sbxConfig struct {
	WebhookUrls []string `json:"webhookUrls"`
	ProductUrls []string `json:"productUrls"`
	ProxyArray  []string `json:"ProxyArray"`
}

type sbxProduct struct {
	URL         string
	Client      *http.Client
	ProductInfo *sbxProductInfo
	FirstRun    bool
	PageRemoved bool
	sync.Mutex
	SizeAvailability map[string]bool
	SizeMap          map[string]string
}

type sbxProductInfo struct {
	ProductName, ProductPrice, ProductImage string
}

type sbxSize struct {
	SizeName, SizeAID string
	Available         bool
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
