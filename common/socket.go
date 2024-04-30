package common

import (
	"context"
	"errors"
	"fmt"
	"github.com/RomiChan/websocket"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type S struct {
	url     string
	proxies string
	headers map[string]string
	query   []string
	err     error
}

func SocketBuilder() *S {
	return &S{
		query:   make([]string, 0),
		headers: make(map[string]string),
	}
}

func (s *S) URL(url string) *S {
	s.url = url
	return s
}

func (s *S) Proxies(proxies string) *S {
	s.proxies = proxies
	return s
}

func (s *S) Header(key, value string) *S {
	s.headers[key] = value
	return s
}

func (s *S) Query(key, value string) *S {
	s.query = append(s.query, fmt.Sprintf("%s=%s", key, value))
	return s
}

func (s *S) Do() (*websocket.Conn, *http.Response, error) {
	if s.err != nil {
		return nil, nil, s.err
	}

	if s.url == "" {
		return nil, nil, errors.New("url cannot be empty, please execute func URL(url string)")
	}

	query := ""
	if len(s.query) > 0 {
		var slice []string
		for _, value := range s.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}

	h := http.Header{}
	for k, v := range s.headers {
		h.Add(k, v)
	}

	return socket(s.proxies, s.url+query, h)
}

func (s *S) DoWith(status int) (*websocket.Conn, error) {
	conn, response, err := s.Do()
	if err != nil {
		return nil, err
	}

	if response.StatusCode != status {
		return nil, errors.New(response.Status)
	}

	return conn, nil
}

func socket(proxies, urlStr string, header http.Header) (*websocket.Conn, *http.Response, error) {
	dialer := websocket.DefaultDialer
	if proxies != "" {
		pu, err := url.Parse(proxies)
		if err != nil {
			return nil, nil, err
		}

		if pu.Scheme == "http" || pu.Scheme == "https" {
			dialer = &websocket.Dialer{
				Proxy:            http.ProxyURL(pu),
				HandshakeTimeout: 45 * time.Second,
			}
		}

		if pu.Scheme == "socks5" {
			dialer = &websocket.Dialer{
				NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d, e := proxy.SOCKS5("tcp", pu.Host, nil, proxy.Direct)
					if e != nil {
						return nil, e
					}
					return d.Dial(network, addr)
				},
				HandshakeTimeout: 45 * time.Second,
			}
		}
	}

	return dialer.Dial(urlStr, header)
}
