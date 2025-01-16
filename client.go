package webpush

import "github.com/i9si-sistemas/nine"

type Client struct {
	httpClient nine.Client
}

func New(httpClient nine.Client) *Client {
	return &Client{httpClient: httpClient}
}
