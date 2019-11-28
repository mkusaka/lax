package client

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

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
	// TODO: get replacement correspondence from db
	// URL.String not returns valid url...?
	requestURL := extractURL(request)
	// fetch replace correspondence & settings (like key element, expire configure etc...) from db
	// TODO: fetch correspondence & cache at once?
	proxyURL := strings.Replace(requestURL, ":300", ":3000", 1)
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

func extractURL(request *http.Request) string {
	host := request.Host
	path := request.RequestURI
	schema := "http"
	if request.TLS != nil {
		schema = "https"
	}

	return schema + "://" + host + path
}
