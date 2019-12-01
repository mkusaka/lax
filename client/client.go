package client

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/mkusaka/lax/db"
)

var dbClient = db.NewClient(1000 * time.Millisecond)

type Client struct {
	client *http.Client

	Logger *log.Logger
}

func NewClient(timeoutSecond time.Duration) *Client {
	client := &http.Client{
		Timeout: timeoutSecond,
	}
	return &Client{
		client: client,
		Logger: log.New(ioutil.Discard, "go-client: ", log.LstdFlags),
	}
}

func (c *Client) ProxyRequest(request *http.Request) (*http.Response, error) {
	// URL.String not returns valid url...?
	config, err := dbClient.GetConfigFromDomain(request.Host)
	if err != nil {
		return nil, err
	}

	proxyPath, err := config.ProxyPath(request.RequestURI)
	// proxyPath := "posts"

	if err != nil {
		return nil, err
	}
	proxyURL := GenerateURL(request, config.ProxyDomain, proxyPath)
	fmt.Printf("proxy url: %s \n", proxyURL)

	key := config.Key(currentSchema(request), request.RequestURI)

	cacheMeta := dbClient.GetCacheMeta(config.ID, key)

	var cache *db.CacheEntity
	if cacheMeta != nil && !cacheMeta.IsExpired() {
		cache = dbClient.GetCacheEntity(cacheMeta.ID)
		return db.GenerateResponseFromCache(cache)
	}
	// fetch cache from db. key: proxyURL/method/vary header or something user defined
	// if cache exists & not error response, construct http.response & return it.

	req, err := http.NewRequest(request.Method, proxyURL, request.Body)
	if err != nil {
		return nil, err
	}

	req.Header = request.Header

	res, err := c.client.Do(req)

	// store response to db.
	// TODO: not store PUT/POST/DELETE by default. user wil configable cache path.

	return res, err
}

func currentSchema(request *http.Request) string {
	if request.TLS != nil {
		return "https"
	}
	return "http"
}

func GenerateURL(request *http.Request, proxyDomain, proxyPath string) string {
	schema := currentSchema(request)

	return schema + "://" + proxyDomain + proxyPath
}
