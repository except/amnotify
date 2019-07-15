package main

import "net/http"

type endProdInfo struct {
	Name, ProductURL, Price, ImageURL string
}

type endTask struct {
	ProductSKU string

	FirstRun bool

	Client      *http.Client
	ProductInfo *endProdInfo

	SizeMap map[string]bool
}

type endProduct struct {
	ID                  int    `json:"id"`
	Sku                 string `json:"sku"`
	Name                string `json:"name"`
	Link                string `json:"link"`
	InStock             bool   `json:"in_stock"`
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
