package irail

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

var rateLimit = rate.NewLimiter(rate.Every(time.Second), 100)

type RLHTTPClient struct {
	client      *http.Client
	Ratelimiter *rate.Limiter
}

func (c *RLHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := c.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		log.Println("Rate limited, waiting 1 second")
		time.Sleep(1 * time.Second)
		return c.Do(req)
	}
	return resp, nil
}

func newClient() *RLHTTPClient {
	c := &RLHTTPClient{
		client:      http.DefaultClient,
		Ratelimiter: rateLimit,
	}
	return c
}
