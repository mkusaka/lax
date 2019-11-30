package client

import (
	"errors"
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
	if err == nil {
		return nil, errors.New("bad request")
	}

	proxyPath, err := config.ProxyPath(request.RequestURI)

	if err != nil {
		return nil, errors.New("invalid rule")
	}
	// fetch replace correspondence & settings (like key element, expire configure etc...) from db
	// TODO: fetch correspondence & cache at once?
	proxyURL := GenerateURL(request, config.ProxyDomain, proxyPath)
	// fetch cache from db. key: proxyURL/method/vary header or something user defined
	// if cache exists & not error response, construct http.response & return it.

	fmt.Printf("proxy url: %s \n", proxyURL)

	req, err := http.NewRequest(request.Method, proxyURL, request.Body)
	if err != nil {
		return nil, err
	}

	req.Header = request.Header

	res, err := c.client.Do(req)

	// store response to db.
	// but not store PUT/POST/DELETE by default. user wil configable cache path.

	return res, err
}

func GenerateURL(request *http.Request, proxyDomain, proxyPath string) string {
	schema := "http"
	if request.TLS != nil {
		schema = "https"
	}

	return schema + "://" + proxyDomain + proxyPath
}
