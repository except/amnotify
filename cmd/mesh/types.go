package main

type meshSiteConfig map[string]*meshSite

type meshSite struct {
	SiteURL   string `json:"SiteURL"`
	SKUSuffix string `json:"SKUSuffix"`
}

type meshConfig []meshConfigProduct

type meshConfigProduct struct {
	SKU   string   `json:"SKU"`
	Sites []string `json:"Sites"`
}

type meshTask struct {
	SKU           string
	Site          *meshSite
	ProductInfo   *meshProductInfo
	ProductSKUMap map[string]*meshProductSKU
}

type meshProductInfo struct {
	Name, Price, ImageURL string
}

type meshWishlist struct {
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

type meshProductSKU struct {
	SKU         string `json:"SKU"`
	StockStatus string `json:"stockStatus"`
}
