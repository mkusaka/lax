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
	fullPath := extractUrl(request)
	newUrl := strings.Replace(fullPath, ":300", ":3000", 1)
	fmt.Printf("proxy url: %s \n", newUrl)

	req, err := http.NewRequest(request.Method, newUrl, request.Body)
	if err != nil {
		return nil, err
	}

	req.Header = request.Header

	return c.client.Do(req)
}

func extractUrl(request *http.Request) string {
	host := request.Host
	path := request.RequestURI
	schema := "http"
	if request.TLS != nil {
		schema = "https"
	}

	return schema + "://" + host + path
}
