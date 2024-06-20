package emit

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
	ctx     context.Context

	jar http.CookieJar

	session *Session
	option  *ConnectOption
}

func NewSocketSession(proxies string, option *ConnectOption) (*Session, error) {
	dialer, err := socket(proxies, option)
	if err != nil {
		return nil, err
	}
	return &Session{
		dialer: dialer,
	}, nil
}

func SocketBuilder(session *Session) *Conn {
	return &Conn{
		query:   make([]string, 0),
		headers: make(map[string]string),
		session: session,
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

func (conn *Conn) Context(ctx context.Context) *Conn {
	conn.ctx = ctx
	return conn
}

func (conn *Conn) CookieJar(jar http.CookieJar) *Conn {
	conn.jar = jar
	return conn
}

func (conn *Conn) Option(opt *ConnectOption) *Conn {
	if opt == nil {
		return conn
	}

	conn.option = opt
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

func (conn *Conn) DoS(status int) (*websocket.Conn, *http.Response, error) {
	return conn.DoC(Status(status))
}

func (conn *Conn) DoC(funs ...func(*http.Response) error) (*websocket.Conn, *http.Response, error) {
	c, response, err := conn.Do()
	if err != nil {
		return c, response, err
	}

	for _, condition := range funs {
		err = condition(response)
		if err != nil {
			_ = response.Body.Close()
			return c, response, err
		}
	}

	return c, response, nil
}

func (conn *Conn) Do() (*websocket.Conn, *http.Response, error) {
	if conn.err != nil {
		return nil, nil, Error{-1, "Do", conn.err}
	}

	if conn.url == "" {
		return nil, nil, Error{-1, "Do", errors.New("url cannot be empty, please execute func URL(url string)")}
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

	var dialer *websocket.Dialer
	if conn.session != nil {
		dialer = conn.session.dialer
	} else {
		var err error
		dialer, err = socket(conn.proxies, conn.option)
		if err != nil {
			err = Error{-1, "Do", err}
		}
	}

	if conn.jar != nil {
		dialer.Jar = conn.jar
	}

	c, response, err := dialer.Dial(conn.url+query, h)
	if err != nil {
		return c, response, err
	}

	if conn.ctx != nil {
		go warpC(c, conn.ctx)
	}

	if response.Request.Body != nil {
		_ = response.Request.Body.Close()
	}

	return c, response, err
}

func socket(proxies string, opts *ConnectOption) (*websocket.Dialer, error) {
	dialer := websocket.DefaultDialer
	if proxies != "" {
		pu, err := url.Parse(proxies)
		if err != nil {
			return nil, err
		}

		handshakeTimeout := 45 * time.Second
		if opts != nil && opts.TLSHandshakeTimeout > 0 {
			handshakeTimeout = opts.TLSHandshakeTimeout
		}

		if pu.Scheme == "http" || pu.Scheme == "https" {
			dialer = &websocket.Dialer{
				Proxy:            http.ProxyURL(pu),
				HandshakeTimeout: handshakeTimeout,
				NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d := &net.Dialer{}
					if opts != nil {
						if opts.IdleConnTimeout > 0 {
							d.KeepAlive = opts.IdleConnTimeout
						}
						if opts.DisableKeepAlive {
							d.KeepAlive = 0
						}
					}
					return d.DialContext(ctx, network, addr)
				},
			}
		}

		if pu.Scheme == "socks5" {
			dialer = &websocket.Dialer{
				NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d, e := proxy.SOCKS5("tcp", pu.Host, nil, proxy.Direct)
					if e != nil {
						return nil, e
					}

					c, e := d.Dial(network, addr)
					if e != nil {
						return nil, e
					}

					conn := c.(*net.TCPConn)
					if opts != nil {
						if opts.IdleConnTimeout > 0 {
							_ = conn.SetKeepAlivePeriod(opts.IdleConnTimeout)
						}
						if opts.DisableKeepAlive {
							_ = conn.SetKeepAlive(false)
						}
					}
					return conn, nil
				},
				HandshakeTimeout: handshakeTimeout,
			}
		}
	}

	return dialer, nil
}

func warpC(c *websocket.Conn, ctx context.Context) {
	if c == nil || ctx == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				_ = c.Close()
			}
			return
		default:
			time.Sleep(time.Second) //
		}
	}
}
