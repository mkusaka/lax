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

	proxyPath, err := config.ProxyPath(request.RequestURI, request.Method)
	// proxyPath := "posts"

	if err != nil {
		return nil, err
	}

	// TODO: move this logic to db or service layer
	proxyURL := GenerateURL(request, config.ProxyDomain, proxyPath)
	fmt.Printf("proxy url: %s \n", proxyURL)

	key := config.Key(currentSchema(request), request.RequestURI, request.Method)

	// fetch cache from db. key: proxyURL/method/vary header or something user defined
	cacheMeta := dbClient.GetCacheMeta(config.ID, key)

	var cache *db.CacheEntity
	if cacheMeta != nil && !cacheMeta.IsExpired() {
		// if cache exists & not error response, construct http.response & return it.
		cache = dbClient.GetCacheEntity(cacheMeta.ID)
		res, err := db.GenerateResponseFromCache(cache)
		if err != nil {
			return nil, err
		}

		// no error code
		if res.StatusCode < 400 {
			return res, nil
		}
	}

	req, err := http.NewRequest(request.Method, proxyURL, request.Body)
	if err != nil {
		// TODO: if error given & response status code >= 400 & cached stale-if-error header, then response cached page
		return nil, err
	}

	req.Header = request.Header

	res, err := c.client.Do(req)

	// store response to db.
	// no cache pattern
	matchedRule := config.MatchRule(request.RequestURI, request.Method)
	if cacheMeta == nil {
		cacheMeta = dbClient.SaveCacheMeta(db.NewCacheMeta(config, matchedRule, key))
		// generate cacheMeta
		// assign to cacheMeta variable
	}

	if cacheMeta.IsExpired() {
		cacheEntity := dbClient.CreateCache(cacheMeta, res)
		dbClient.UpdateCacheMeta(cacheMeta.ID, cacheEntity, config.ExpireAt(matchedRule))
		// generate cacheEntity (cacheEntity will deleted by mongodb's ttl if expired & cacheentity exist)
	}
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
