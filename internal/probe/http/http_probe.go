package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPProbe struct {
	Region                  string
	Endpoint                string
	Method                  string
	Body                    string
	Headers                 map[string]string
	ProxyURL                string
	UnacceptableStatusCodes []int
	Timeout                 time.Duration
	SkipTLSVerify           bool
}

func NewHTTPProbe(endpoint, method, body, proxyURL string, headers map[string]string, unacceptableStatusCodes []int, timeout time.Duration, skipTLSVerify bool) *HTTPProbe {
	return &HTTPProbe{
		Endpoint:                endpoint,
		Method:                  method,
		Body:                    body,
		Headers:                 headers,
		ProxyURL:                proxyURL,
		UnacceptableStatusCodes: unacceptableStatusCodes,
		Timeout:                 timeout,
		SkipTLSVerify:           skipTLSVerify,
		Region:                  "",
	}
}

func (p *HTTPProbe) Probe(ctx context.Context) error {
	client := &http.Client{
		Timeout: p.Timeout,
	}

	transport := &http.Transport{}
	if p.ProxyURL != "" {
		proxy, err := url.Parse(p.ProxyURL)
		if err != nil {
			return err
		}
		transport.Proxy = http.ProxyURL(proxy)
	}
	if p.SkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client.Transport = transport

	var body io.Reader
	if p.Body != "" {
		body = strings.NewReader(p.Body)
	}

	req, err := http.NewRequestWithContext(ctx, p.Method, p.Endpoint, body)
	if err != nil {
		return err
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for _, code := range p.UnacceptableStatusCodes {
		if resp.StatusCode == code {
			return &HTTPStatusError{StatusCode: resp.StatusCode}
		}
	}
	return nil
}

func (p *HTTPProbe) MetadataString() string {
	return fmt.Sprintf("Endpoint: %s | Method: %s | Proxy: %s | Region: %s", p.Endpoint, p.Method, p.ProxyURL, p.Region)
}

type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return "unexpected status code: " + http.StatusText(e.StatusCode)
}
