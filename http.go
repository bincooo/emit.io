package emit

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RomiChan/websocket"
	"github.com/gospider007/ja3"
	"github.com/gospider007/requests"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ConnectOption struct {
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration
	IdleConnTimeout       time.Duration
	DisableKeepAlive      bool
	MaxIdleConnects       int

	TLSClientConfig *tls.Config
}

type Client struct {
	url     string
	method  string
	proxies string
	headers map[string]string
	query   []string
	bytes   []byte
	err     error
	ctx     context.Context
	buffer  io.Reader
	jar     http.CookieJar
	ja3     string
	session *Session
	option  *ConnectOption
	whites  []string
}

type Session struct {
	client   *http.Client
	requests *requests.Client
	dialer   *websocket.Dialer
}

func (session *Session) IdleClose() {
	if session.client != nil {
		session.client.CloseIdleConnections()
	}
	if session.requests != nil {
		session.requests.CloseConns()
	}
}

func NewDefaultSession(proxies string, option *ConnectOption, whites ...string) (*Session, error) {
	cli, err := client(proxies, whites, option)
	if err != nil {
		return nil, err
	}

	return &Session{
		client: cli,
	}, nil
}

func NewJa3Session(proxies string) (*Session, error) {
	s, err := requests.NewClient(context.Background())
	if err != nil {
		return nil, Error{-1, "Do", err}
	}

	err = s.SetProxy(proxies)
	if err != nil {
		return nil, Error{-1, "Do", err}
	}
	return &Session{
		requests: s,
	}, nil
}

func MergeSession(sessions ...*Session) (session *Session) {
	for _, s := range sessions {
		if s == nil {
			continue
		}

		if session == nil {
			session = s
			continue
		}

		if s.client != nil {
			session.client = s.client
		}

		if s.requests != nil {
			session.requests = s.requests
		}

		if s.dialer != nil {
			session.dialer = s.dialer
		}
	}
	return
}

func ClientBuilder(session *Session) *Client {
	return &Client{
		method:  http.MethodGet,
		query:   make([]string, 0),
		headers: make(map[string]string),
		session: session,
	}
}

func (c *Client) URL(url string) *Client {
	c.url = url
	return c
}

func (c *Client) Method(method string) *Client {
	c.method = method
	return c
}

func (c *Client) GET(url string) *Client {
	c.url = url
	c.method = http.MethodGet
	return c
}

func (c *Client) POST(url string) *Client {
	c.url = url
	c.method = http.MethodPost
	return c
}

func (c *Client) PUT(url string) *Client {
	c.url = url
	c.method = http.MethodPut
	return c
}

func (c *Client) DELETE(url string) *Client {
	c.url = url
	c.method = http.MethodDelete
	return c
}

func (c *Client) Proxies(proxies string, whites ...string) *Client {
	c.proxies = proxies
	c.whites = whites
	return c
}

func (c *Client) Context(ctx context.Context) *Client {
	c.ctx = ctx
	return c
}

func (c *Client) CookieJar(jar http.CookieJar) *Client {
	c.jar = jar
	return c
}

func (c *Client) Option(opt *ConnectOption) *Client {
	if opt == nil {
		return c
	}

	if opt.TLSClientConfig == nil && c.option != nil {
		opt.TLSClientConfig = c.option.TLSClientConfig
	}

	c.option = opt
	return c
}

func (c *Client) JHeader() *Client {
	c.headers["content-type"] = "application/json"
	return c
}

func (c *Client) Header(key, value string) *Client {
	c.headers[key] = value
	return c
}

func (c *Client) Query(key, value string) *Client {
	c.query = append(c.query, fmt.Sprintf("%s=%s", key, value))
	return c
}

func (c *Client) Ja3(ja3 string) *Client {
	c.ja3 = ja3
	return c
}

func (c *Client) Body(payload interface{}) *Client {
	if c.err != nil {
		return c
	}
	c.bytes, c.err = json.Marshal(payload)
	return c
}

func (c *Client) Bytes(data []byte) *Client {
	c.bytes = data
	return c
}

func (c *Client) Buffer(buffer io.Reader) *Client {
	c.buffer = buffer
	return c
}

func (c *Client) DoS(status int) (*http.Response, error) {
	return c.DoC(Status(status))
}

func (c *Client) DoC(funs ...func(*http.Response) error) (*http.Response, error) {
	response, err := c.Do()
	if err != nil {
		return response, err
	}

	for _, condition := range funs {
		err = condition(response)
		if err != nil {
			_ = response.Body.Close()
			return response, err
		}
	}

	return response, nil
}

func (c *Client) Do() (*http.Response, error) {
	if c.err != nil {
		return nil, c.err
	}

	if c.url == "" {
		return nil, Error{
			Code: -1,
			Bus:  "Do",
			Err:  errors.New("url cannot be empty, please execute func URL(url string)"),
		}
	}

	if c.ja3 != "" {
		return c.doJ()
	}

	var session *http.Client
	if c.session != nil && c.session.client != nil {
		session = c.session.client
	} else {
		cli, err := client(c.proxies, c.whites, c.option)
		if err != nil {
			return nil, Error{-1, "Do", err}
		}
		session = cli
	}

	query := ""
	if len(c.query) > 0 {
		var slice []string
		for _, value := range c.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}

	if c.jar != nil {
		session.Jar = c.jar
	}

	if c.buffer == nil {
		c.buffer = bytes.NewBuffer(c.bytes)
	}

	request, err := http.NewRequest(c.method, c.url+query, c.buffer)
	if err != nil {
		return nil, Error{-1, "Do", err}
	}

	h := request.Header
	for k, v := range c.headers {
		h.Add(k, v)
	}

	if c.ctx != nil {
		request = request.WithContext(c.ctx)
	}

	response, err := session.Do(request)
	if err != nil {
		err = Error{-1, "Do", err}
	}

	_ = request.Body.Close()
	return response, err
}

func (c *Client) doJ() (*http.Response, error) {
	if c.err != nil {
		return nil, c.err
	}

	if c.url == "" {
		return nil, Error{
			Code: -1,
			Bus:  "Do",
			Err:  errors.New("url cannot be empty, please execute func URL(url string)"),
		}
	}

	if c.ctx == nil {
		var cancel context.CancelFunc
		c.ctx, cancel = context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
	}

	var session *requests.Client
	if c.session != nil && c.session.requests != nil {
		session = c.session.requests
	} else {
		s, err := requests.NewClient(c.ctx)
		if err != nil {
			return nil, Error{-1, "Do", err}
		}

		err = s.SetProxy(c.proxies)
		if err != nil {
			return nil, Error{-1, "Do", err}
		}
		session = s
	}

	query := ""
	if len(c.query) > 0 {
		var slice []string
		for _, value := range c.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}

	if c.jar != nil {
		var (
			u   *url.URL
			err error
		)

		if u, err = url.Parse(c.url); err != nil {
			return nil, Error{-1, "Do", err}
		}

		if err = session.SetCookies(u, c.jar); err != nil {
			return nil, Error{-1, "Do", err}
		}
	}

	if c.buffer == nil {
		c.buffer = bytes.NewBuffer(c.bytes)
	}

	h := requests.NewOrderMap()
	for k, v := range c.headers {
		h.Set(k, v)
	}

	ja3Spec, err := ja3.CreateSpecWithStr(c.ja3) //create ja3 spec with string
	if err != nil {
		return nil, Error{-1, "Do", err}
	}

	response, err := session.Request(c.ctx, c.method, c.url+query, requests.RequestOption{
		Headers: h,
		Ja3Spec: ja3Spec,
		Body:    c.buffer,
	})
	if err != nil {
		return nil, Error{-1, "Do", err}
	}

	var body io.ReadCloser
	if response.IsStream() {
		body = response.Conn()
	} else {
		body = io.NopCloser(bytes.NewBuffer(response.Content()))
		response.CloseBody()
	}

	protoMajor, _ := strconv.Atoi(response.Proto()[5:6])
	r := http.Response{
		Status:        response.Status(),
		StatusCode:    response.StatusCode(),
		Proto:         response.Proto(),
		ProtoMajor:    protoMajor,
		Header:        response.Headers(),
		ContentLength: response.ContentLength(),
		Body:          body,
	}
	return &r, err
}

func client(proxies string, whites []string, option *ConnectOption) (*http.Client, error) {
	c := http.DefaultClient

	newTransport := func(t *http.Transport) http.RoundTripper {
		if option == nil {
			return t
		}

		if t == nil {
			t = &http.Transport{}
		}

		if option.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			t.TLSClientConfig = option.TLSClientConfig
		}

		t.TLSHandshakeTimeout = option.TLSHandshakeTimeout
		t.ResponseHeaderTimeout = option.ResponseHeaderTimeout
		t.ExpectContinueTimeout = option.ExpectContinueTimeout
		t.IdleConnTimeout = option.IdleConnTimeout
		t.MaxIdleConns = option.MaxIdleConnects
		t.DisableKeepAlives = option.DisableKeepAlive

		return t
	}

	if proxies != "" {
		proxiesUrl, err := url.Parse(proxies)
		if err != nil {
			return nil, err
		}

		if proxiesUrl.Scheme == "http" || proxiesUrl.Scheme == "https" {
			c = &http.Client{
				Transport: newTransport(&http.Transport{
					Proxy: func(r *http.Request) (*url.URL, error) {
						if r.URL != nil {
							for _, w := range whites {
								if strings.HasPrefix(r.URL.Host, w) {
									return r.URL, nil
								}
							}
						}
						return proxiesUrl, nil
					},
				}),
			}
		}

		// socks5://127.0.0.1:7890
		if proxiesUrl.Scheme == "socks5" {
			c = &http.Client{
				Transport: newTransport(&http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						for _, w := range whites {
							if strings.HasPrefix(addr, w) {
								return proxy.Direct.Dial(network, addr)
							}
						}

						dialer, e := proxy.SOCKS5("tcp", proxiesUrl.Host, nil, proxy.Direct)
						if e != nil {
							return nil, e
						}
						return dialer.Dial(network, addr)
					},
				}),
			}
		}
	} else if c.Transport == nil {
		c.Transport = newTransport(nil)
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

func TextResponse(response *http.Response) (value string) {
	if response == nil {
		return
	}
	bin, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	return string(bin)
}
