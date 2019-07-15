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
