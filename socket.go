package emits

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

type Conn struct {
	url     string
	proxies string
	headers map[string]string
	query   []string
	err     error
}

func SocketBuilder() *Conn {
	return &Conn{
		query:   make([]string, 0),
		headers: make(map[string]string),
	}
}

func (conn *Conn) URL(url string) *Conn {
	conn.url = url
	return conn
}

func (conn *Conn) Proxies(proxies string) *Conn {
	conn.proxies = proxies
	return conn
}

func (conn *Conn) Header(key, value string) *Conn {
	conn.headers[key] = value
	return conn
}

func (conn *Conn) Query(key, value string) *Conn {
	conn.query = append(conn.query, fmt.Sprintf("%s=%s", key, value))
	return conn
}

func (conn *Conn) Do() (*websocket.Conn, *http.Response, error) {
	if conn.err != nil {
		return nil, nil, conn.err
	}

	if conn.url == "" {
		return nil, nil, errors.New("url cannot be empty, please execute func URL(url string)")
	}

	query := ""
	if len(conn.query) > 0 {
		var slice []string
		for _, value := range conn.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}

	h := http.Header{}
	for k, v := range conn.headers {
		h.Add(k, v)
	}

	return socket(conn.proxies, conn.url+query, h)
}

func (conn *Conn) DoWith(status int) (*websocket.Conn, error) {
	c, response, err := conn.Do()
	if err != nil {
		return nil, err
	}

	if response.StatusCode != status {
		return nil, errors.New(response.Status)
	}

	return c, nil
}

func socket(proxies, u string, header http.Header) (*websocket.Conn, *http.Response, error) {
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

	return dialer.Dial(u, header)
}
