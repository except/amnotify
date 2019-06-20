package main

type meshConfiguation map[string]meshSite

type meshSite struct {
	SiteURL      string   `json:"SiteURL"`
	SKUArray     []string `json:"SKUArray"`
	WebhookArray []string `json:"WebhookArray"`
}
