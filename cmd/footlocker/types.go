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
	BaseURL        string   `json:"BaseUrl"`
	WebhookUrls    []string `json:"WebhookUrls"`
	CurrencySymbol string   `json:"CurrencySymbol"`
}

type ftlContent struct {
	Content string `json:"content"`
}

type ftlTask struct {
	SKU         string
	FirstRun    bool
	PageRemoved bool

	Region      *ftlRegion
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
