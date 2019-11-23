package main

import (
	"log"
	"math/rand"
	"time"
)

func (c *endCookies) GetCookieSet(productSKU string) string {
	cookie := cookieArray[rand.Intn(len(cookieArray))]

	cookieExpiry := c.GetCookieRefresh(cookie)
	if time.Now().Sub(cookieExpiry) > 0 {
		c.UpdateCookieRefresh(cookie)
		return cookie
	}

	log.Printf("[WARN] Failed Fetching Cookie Set - %v", productSKU)
	return c.GetCookieSet(productSKU)
}

func (c *endCookies) GetCookieRefresh(cookie string) time.Time {
	defer c.mu.Unlock()
	c.mu.Lock()

	return c.Map[cookie]
}

func (c *endCookies) UpdateCookieRefresh(cookie string) {
	defer c.mu.Unlock()
	c.mu.Lock()

	c.Map[cookie] = time.Now().Add(30 * time.Minute)
}
