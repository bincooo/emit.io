package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type R struct {
	url     string
	method  string
	proxies string
	headers map[string]string
	query   []string
	bytes   []byte
	err     error
	ctx     context.Context
}

func ClientBuilder() *R {
	return &R{
		method:  http.MethodGet,
		query:   make([]string, 0),
		headers: make(map[string]string),
	}
}

func (r *R) URL(url string) *R {
	r.url = url
	return r
}

func (r *R) Method(method string) *R {
	r.method = method
	return r
}

func (r *R) GET(url string) *R {
	r.url = url
	r.method = http.MethodGet
	return r
}

func (r *R) POST(url string) *R {
	r.url = url
	r.method = http.MethodPost
	return r
}

func (r *R) PUT(url string) *R {
	r.url = url
	r.method = http.MethodPut
	return r
}

func (r *R) DELETE(url string) *R {
	r.url = url
	r.method = http.MethodDelete
	return r
}

func (r *R) Proxies(proxies string) *R {
	r.proxies = proxies
	return r
}

func (r *R) Context(ctx context.Context) *R {
	r.ctx = ctx
	return r
}

func (r *R) JHeader() *R {
	r.headers["content-type"] = "application/json"
	return r
}

func (r *R) Header(key, value string) *R {
	r.headers[key] = value
	return r
}

func (r *R) Query(key, value string) *R {
	r.query = append(r.query, fmt.Sprintf("%s=%s", key, value))
	return r
}

func (r *R) Body(payload interface{}) *R {
	if r.err != nil {
		return r
	}
	r.bytes, r.err = json.Marshal(payload)
	return r
}

func (r *R) Bytes(data []byte) *R {
	r.bytes = data
	return r
}

func (r *R) DoWith(status int) (*http.Response, error) {
	response, err := r.Do()
	if err != nil {
		return nil, err
	}

	if response.StatusCode != status {
		return nil, errors.New(response.Status)
	}

	return response, nil
}

func (r *R) Do() (*http.Response, error) {
	if r.err != nil {
		return nil, r.err
	}

	if r.url == "" {
		return nil, errors.New("url cannot be empty, please execute func URL(url string)")
	}

	c, err := client(r.proxies)
	if err != nil {
		return nil, err
	}

	query := ""
	if len(r.query) > 0 {
		var slice []string
		for _, value := range r.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}
	request, err := http.NewRequest(r.method, r.url+query, bytes.NewBuffer(r.bytes))
	if err != nil {
		return nil, err
	}

	h := request.Header
	for k, v := range r.headers {
		h.Add(k, v)
	}

	if r.ctx != nil {
		request = request.WithContext(r.ctx)
	}

	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func client(proxies string) (*http.Client, error) {
	c := http.DefaultClient
	if proxies != "" {
		proxiesUrl, err := url.Parse(proxies)
		if err != nil {
			return nil, err
		}

		if proxiesUrl.Scheme == "http" || proxiesUrl.Scheme == "https" {
			c = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxiesUrl),
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}

		// socks5://127.0.0.1:7890
		if proxiesUrl.Scheme == "socks5" {
			c = &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						dialer, e := proxy.SOCKS5("tcp", proxiesUrl.Host, nil, proxy.Direct)
						if e != nil {
							return nil, e
						}
						return dialer.Dial(network, addr)
					},
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
	}

	return c, nil
}

func ToObject(response *http.Response, obj interface{}) (err error) {
	var data []byte
	data, err = io.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, obj)
	return
}

func ToMap(response *http.Response) (obj map[string]interface{}, err error) {
	err = ToObject(response, &obj)
	return
}

func ToSlice(response *http.Response) (slice []map[string]interface{}, err error) {
	err = ToObject(response, &slice)
	return
}

func GetCookie(response *http.Response, key string) string {
	cookies := response.Header.Values("set-cookie")
	if len(cookies) == 0 {
		return ""
	}

	for _, cookie := range cookies {
		if !strings.HasPrefix(cookie, key+"=") {
			continue
		}

		cookie = strings.TrimPrefix(cookie, key+"=")
		cos := strings.Split(cookie, "; ")
		if len(cos) > 0 {
			return cos[0]
		}
	}

	return ""
}

func GetCookies(response *http.Response) string {
	cookies := response.Header.Values("set-cookie")
	if len(cookies) == 0 {
		return ""
	}

	var buffer []string
	for _, cookie := range cookies {
		cos := strings.Split(cookie, "; ")
		if len(cos) > 0 {
			buffer = append(buffer, cos[0])
		}
	}

	return strings.Join(buffer, "; ")
}

func MergeCookies(sourceCookies, targetCookies string) string {
	_sourceCookies := strings.Split(sourceCookies, "; ")
	_targetCookies := strings.Split(targetCookies, "; ")

	cached := make(map[string]string)
	for _, cookie := range _sourceCookies {
		kv := strings.Split(cookie, "=")
		if len(kv) < 2 {
			continue
		}

		k := strings.TrimSpace(kv[0])
		cached[k] = strings.Join(kv[1:], "=")
	}

	for _, cookie := range _targetCookies {
		kv := strings.Split(cookie, "=")
		if len(kv) < 2 {
			continue
		}

		k := strings.TrimSpace(kv[0])
		if len(k) == 0 {
			continue
		}

		cached[k] = strings.Join(kv[1:], "=")
	}

	var buffer []string
	for k, v := range cached {
		buffer = append(buffer, k+"="+v)
	}

	return strings.Join(buffer, "; ")
}
