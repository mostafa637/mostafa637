package zai

import (
	"net/http"
	"strings"
)

type Client struct {
	config Config
	http   *http.Client
}

func New(config Config, client *http.Client) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{config: config, http: client}, nil
}

func NewFromEnv(client *http.Client) (*Client, error) {
	return New(ConfigFromEnv(), client)
}

func (c *Client) endpoint() string {
	return strings.TrimRight(c.config.BaseURL, "/") + "/chat/completions"
}

func (c *Client) model() string {
	return c.config.Model
}
