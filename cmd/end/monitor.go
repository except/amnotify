package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dchest/uniuri"
)

var (
	errTaskBanned       = errors.New("Task Banned")
	errProductOOS       = errors.New("Product is out of stock")
	errProductNoSizes   = errors.New("Product has no available sizes")
	errProductNotLoaded = errors.New("Product not loaded")
)

func (t *endTask) Monitor() {

}

func (t *endTask) GetSizes() (map[string]bool, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://admin2.endclothing.com/gb/rest/V1/end/products/sku/%v?/%v=%v", t.ProductSKU, uniuri.NewLen(8), uniuri.NewLen(8)), nil)

	if err != nil {
		return nil, err
	}

	resp, err := t.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		var product endProduct
		err = json.NewDecoder(resp.Body).Decode(&product)

		if err != nil {
			return nil, err
		}

		if t.ProductInfo != nil {
			prodInfo := &endProdInfo{
				Name:       product.Name,
				ProductURL: product.Link,
				Price:      fmt.Sprintf("£%v", product.Price),
			}

			if len(product.MediaGalleryEntries) > 0 {
				prodInfo.ImageURL = product.MediaGalleryEntries[0].File
			}

			t.ProductInfo = prodInfo
		}

		if product.InStock {
			sizesAvailable := false
			sizeMap := make(map[string]bool)
			for _, sizeOption := range product.Options {
				if sizeOption.AttributeID == "173" && sizeOption.Label == "Size" {
					if len(sizeOption.Values) > 0 {
						sizesAvailable = true
						for _, individualSize := range sizeOption.Values {
							sizeMap[individualSize.Label] = individualSize.InStock
						}
					}
				}
			}
			if sizesAvailable {
				return sizeMap, nil
			}
			return nil, errProductNoSizes
		}
		return nil, errProductOOS
	case 404:
		return nil, errProductNotLoaded
	case 403:
		return nil, errTaskBanned
	case 456:
		return nil, errTaskBanned
	default:
		return nil, fmt.Errorf("Invalid Status Code - %v", resp.StatusCode)
	}
}
