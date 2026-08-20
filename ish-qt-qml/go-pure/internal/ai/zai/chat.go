package zai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) Chat(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if request.Model == "" {
		request.Model = c.model()
	}
	body, err := json.Marshal(request)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("zai: encode request: %w", err)
	}
	result, err := c.do(ctx, body)
	if err != nil {
		return ChatResponse{}, err
	}
	return result, nil
}

func (c *Client) do(ctx context.Context, body []byte) (ChatResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("zai: create request: %w", err)
	}
	setHeaders(req, c.config.APIKey)
	res, err := c.http.Do(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("zai: request: %w", err)
	}
	defer res.Body.Close()
	return decodeResponse(res)
}

func setHeaders(req *http.Request, key string) {
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func decodeResponse(res *http.Response) (ChatResponse, error) {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("zai: read response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return ChatResponse{}, fmt.Errorf("zai: HTTP %s: %s", res.Status, string(data))
	}
	var result ChatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return ChatResponse{}, fmt.Errorf("zai: decode response: %w", err)
	}
	return result, nil
}
