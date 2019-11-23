package main

import (
	"log"
	"math/rand"
	"time"
)

func (c *endCookies) GetCookieSet(productSKU string) string {
	cookie := cookies.CookieArray[rand.Intn(len(c.CookieArray))]

	defer c.mu.Unlock()

	c.mu.Lock()

	if time.Now().Sub(c.Map[cookie]) > 0 {
		c.Map[cookie] = time.Now().Add(30 * time.Minute)
		return cookie
	}

	log.Printf("[WARN] Failed Fetching Cookie Set - %v", productSKU)
	return c.GetCookieSet(productSKU)
}
