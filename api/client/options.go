package client

type Option func(*Client) error

func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		if baseURL == "" {
			return ErrInvalidBaseURL
		}
		c.baseURL = baseURL
		return nil
	}
}

func WithHTTPClient(httpClient HTTPClient) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return ErrInvalidHTTPClient
		}
		c.httpClient = httpClient
		return nil
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}
