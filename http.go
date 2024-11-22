package emit

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RomiChan/websocket"
	"github.com/andybalholm/brotli"
	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strings"
	"time"
)

type Wip func() []string

type Echo struct {
	RandomTLSExtension bool
	HelloID            profiles.ClientProfile
}

type ConnectOption struct {
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	expectContinueTimeout time.Duration
	idleConnTimeout       time.Duration
	disableKeepAlive      bool
	maxIdleConnects       int

	tlsConfig *tls.Config
}

type Builder struct {
	url         string
	method      string
	proxies     string
	headers     map[string]string
	query       []string
	bytes       []byte
	err         error
	ctx         context.Context
	buffer      io.Reader
	jar         http.CookieJar
	ja3         string
	session     *Session
	option      *ConnectOption
	fetchWithes func() []string

	encoding []string
}

type Session struct {
	opts *ConnectOption

	client    *http.Client
	tlsClient tls_client.HttpClient
	dialer    *websocket.Dialer
}

type OptionHelper = func(string, *Session) error

type readCloser struct {
	io.Reader
	io.Closer
}

func TLSHandshakeTimeoutHelper(timeout time.Duration) OptionHelper {
	return func(_ string, session *Session) error {
		if session.opts == nil {
			session.opts = &ConnectOption{}
		}
		session.opts.tlsHandshakeTimeout = timeout
		return nil
	}
}

func ResponseHeaderTimeoutHelper(timeout time.Duration) OptionHelper {
	return func(_ string, session *Session) error {
		if session.opts == nil {
			session.opts = &ConnectOption{}
		}
		session.opts.responseHeaderTimeout = timeout
		return nil
	}
}

func ExpectContinueTimeoutHelper(timeout time.Duration) OptionHelper {
	return func(_ string, session *Session) error {
		if session.opts == nil {
			session.opts = &ConnectOption{}
		}
		session.opts.expectContinueTimeout = timeout
		return nil
	}
}

func IdleConnTimeoutHelper(timeout time.Duration) OptionHelper {
	return func(_ string, session *Session) error {
		if session.opts == nil {
			session.opts = &ConnectOption{}
		}
		session.opts.idleConnTimeout = timeout
		return nil
	}
}

func DisableKeepAliveHelper(flag bool) OptionHelper {
	return func(_ string, session *Session) error {
		if session.opts == nil {
			session.opts = &ConnectOption{}
		}
		session.opts.disableKeepAlive = flag
		return nil
	}
}

func MaxIdleConnectsHelper(count int) OptionHelper {
	return func(_ string, session *Session) error {
		if session.opts == nil {
			session.opts = &ConnectOption{}
		}
		session.opts.maxIdleConnects = count
		return nil
	}
}

func TLSConfigHelper(config *tls.Config) OptionHelper {
	return func(_ string, session *Session) error {
		if config == nil {
			if session.opts == nil {
				session.opts = &ConnectOption{}
			}
			session.opts.tlsConfig = config
		}
		return nil
	}
}

func Ja3Helper(echo Echo, timeout int) OptionHelper {
	return func(proxies string, session *Session) error {
		jar := tls_client.NewCookieJar()
		options := []tls_client.HttpClientOption{
			tls_client.WithTimeoutSeconds(timeout),
			tls_client.WithClientProfile(echo.HelloID),
			tls_client.WithNotFollowRedirects(),
			tls_client.WithCookieJar(jar),
		}

		if proxies != "" {
			options = append(options, tls_client.WithProxyUrl(proxies))
		}

		if echo.RandomTLSExtension {
			options = append(options, tls_client.WithRandomTLSExtensionOrder())
		}

		c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
		if err != nil {
			return err
		}

		session.tlsClient = c
		return nil
	}
}

func NewSession(proxies string, withes Wip, opts ...OptionHelper) (session *Session, err error) {
	session = &Session{}
	for _, exec := range opts {
		if err = exec(proxies, session); err != nil {
			return
		}
	}

	if withes == nil {
		withes = func() (_ []string) { return }
	}

	c, err := client(proxies, withes, session.opts)
	if err != nil {
		return
	}
	session.client = c

	dialer, err := socket(proxies, withes, session.opts)
	if err != nil {
		return
	}
	session.dialer = dialer

	return
}

func (session *Session) IdleClose() {
	if session.client != nil {
		session.client.CloseIdleConnections()
	}
	if session.tlsClient != nil {
		session.tlsClient.CloseIdleConnections()
	}
}

func ClientBuilder(session *Session) *Builder {
	return &Builder{
		method:      http.MethodGet,
		query:       make([]string, 0),
		headers:     make(map[string]string),
		fetchWithes: func() []string { return nil },
		session:     session,
	}
}

func (c *Builder) URL(url string) *Builder {
	c.url = url
	return c
}

func (c *Builder) Method(method string) *Builder {
	c.method = method
	return c
}

func (c *Builder) GET(url string) *Builder {
	c.url = url
	c.method = http.MethodGet
	return c
}

func (c *Builder) POST(url string) *Builder {
	c.url = url
	c.method = http.MethodPost
	return c
}

func (c *Builder) PUT(url string) *Builder {
	c.url = url
	c.method = http.MethodPut
	return c
}

func (c *Builder) DELETE(url string) *Builder {
	c.url = url
	c.method = http.MethodDelete
	return c
}

func (c *Builder) Proxies(proxies string, whites ...string) *Builder {
	c.proxies = proxies
	c.fetchWithes = func() []string {
		return whites
	}
	return c
}

func (c *Builder) Context(ctx context.Context) *Builder {
	c.ctx = ctx
	return c
}

func (c *Builder) CookieJar(jar http.CookieJar) *Builder {
	c.jar = jar
	return c
}

func (c *Builder) Option(opt *ConnectOption) *Builder {
	if opt == nil {
		return c
	}

	if opt.tlsConfig == nil && c.option != nil {
		opt.tlsConfig = c.option.tlsConfig
	}

	c.option = opt
	return c
}

func (c *Builder) JHeader() *Builder {
	c.headers["content-type"] = "application/json"
	return c
}

func (c *Builder) Header(key, value string) *Builder {
	if key == "" {
		return c
	}
	c.headers[key] = value
	return c
}

func (c *Builder) Query(key, value string) *Builder {
	if key == "" {
		return c
	}
	c.query = append(c.query, fmt.Sprintf("%s=%s", key, value))
	return c
}

func (c *Builder) Ja3() *Builder {
	c.ja3 = "yes"
	return c
}

func (c *Builder) Body(payload interface{}) *Builder {
	if c.err != nil {
		return c
	}
	c.bytes, c.err = json.Marshal(payload)
	return c
}

func (c *Builder) Bytes(data []byte) *Builder {
	c.bytes = data
	return c
}

func (c *Builder) Buffer(buffer io.Reader) *Builder {
	c.buffer = buffer
	return c
}

func (c *Builder) Encoding(encoding ...string) *Builder {
	c.encoding = append(c.encoding, encoding...)
	return c
}

func (c *Builder) DoS(status int) (*http.Response, error) {
	return c.DoC(Status(status))
}

func (c *Builder) DoC(funs ...func(*http.Response) error) (*http.Response, error) {
	response, err := c.Do()
	if err != nil {
		return response, err
	}

	for _, condition := range funs {
		err = condition(response)
		if err != nil {
			if response != nil {
				value := TextResponse(response)
				response.Body = io.NopCloser(bytes.NewBufferString(value))
				_ = response.Body.Close()
			}
			return response, err
		}
	}

	return response, nil
}

func (c *Builder) Do() (*http.Response, error) {
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
		cli, err := client(c.proxies, c.fetchWithes, c.option)
		if err != nil {
			return nil, Error{-1, "Do", "", err}
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
		return nil, Error{-1, "Do", "", err}
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
		err = Error{-1, "Do", "", err}
	}

	if len(c.encoding) > 0 {
		encoding := response.Header.Get("Content-Encoding")
		switch encoding {
		case "gzip":
			if slices.Contains(c.encoding, "gzip") {
				response.Body, err = decodeGZip(response.Body)
				if err != nil {
					return response, Error{-1, "Do decoding/gzip", "", err}
				}
			}
		case "deflate":
			if slices.Contains(c.encoding, "deflate") {
				response.Body = flate.NewReader(response.Body)
			}
		case "br":
			if slices.Contains(c.encoding, "br") {
				response.Body = &readCloser{brotli.NewReader(response.Body), response.Body}
			}
		}
	}

	_ = request.Body.Close()
	return response, err
}

func (c *Builder) doJ() (*http.Response, error) {
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

	query := ""
	if len(c.query) > 0 {
		var slice []string
		for _, value := range c.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}

	if len(c.bytes) == 0 && c.buffer != nil {
		data, err := io.ReadAll(c.buffer)
		if err != nil {
			return nil, Error{-1, "Do", "", err}
		}
		c.bytes = data
	}

	request, err := fhttp.NewRequest(c.method, c.url+query, bytes.NewReader(c.bytes))
	if err != nil {
		return nil, Error{-1, "Do", "", err}
	}

	if c.jar != nil {
		var u *url.URL

		u, err = url.Parse(c.url)
		if err != nil {
			return nil, Error{-1, "Do", "", err}
		}

		cookies := c.jar.Cookies(u)
		var newCookies []*fhttp.Cookie
		for _, cookie := range cookies {
			newCookies = append(newCookies, &fhttp.Cookie{
				Name:  cookie.Name,
				Value: cookie.Value,
			})
		}
		c.session.tlsClient.SetCookies(u, newCookies)
	}

	request.Header = fhttp.Header{}
	for k, v := range c.headers {
		request.Header.Set(k, v)
	}

	response, err := c.session.tlsClient.Do(request)
	if err != nil {
		return nil, Error{-1, "Do", "", err}
	}

	headers := response.Header
	newHeaders := http.Header{}
	for k, _ := range headers {
		newHeaders[k] = headers[k]
	}

	r := http.Response{
		Status:           response.Status,
		StatusCode:       response.StatusCode,
		Proto:            response.Proto,
		ProtoMajor:       response.ProtoMajor,
		ProtoMinor:       response.ProtoMinor,
		Header:           newHeaders,
		Body:             response.Body,
		ContentLength:    response.ContentLength,
		TransferEncoding: response.TransferEncoding,
		Close:            response.Close,
		Uncompressed:     response.Uncompressed,
		Trailer:          (map[string][]string)(response.Trailer),
	}

	return &r, err
}

func client(proxies string, withes Wip, option *ConnectOption) (*http.Client, error) {
	c := http.DefaultClient

	newTransport := func(t *http.Transport) http.RoundTripper {
		if t == nil {
			t = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}

		if option == nil {
			return t
		}

		if option.tlsConfig != nil {
			t.TLSClientConfig = option.tlsConfig
		}

		t.TLSHandshakeTimeout = option.tlsHandshakeTimeout
		t.ResponseHeaderTimeout = option.responseHeaderTimeout
		t.ExpectContinueTimeout = option.expectContinueTimeout
		t.IdleConnTimeout = option.idleConnTimeout
		t.MaxIdleConns = option.maxIdleConnects
		t.DisableKeepAlives = option.disableKeepAlive

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
							for _, w := range withes() {
								if strings.HasPrefix(r.URL.Host, w) {
									return r.URL, nil
								}
							}
						}
						if proxiesUrl.User != nil {
							if u := proxiesUrl.User.Username(); u != "" {
								p, _ := proxiesUrl.User.Password()
								u += ":" + p
								r.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(u)))
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
						for _, w := range withes() {
							if strings.HasPrefix(addr, w) {
								return proxy.Direct.Dial(network, addr)
							}
						}

						var au *proxy.Auth
						if proxiesUrl.User != nil {
							if u := proxiesUrl.User.Username(); u != "" {
								p, _ := proxiesUrl.User.Password()
								au = &proxy.Auth{User: u, Password: p}
							}
						}
						dialer, e := proxy.SOCKS5("tcp", proxiesUrl.Host, au, proxy.Direct)
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

	// encoding := response.Header.Get("Content-Encoding")
	// if encoding != "" && response.Proto == "JA3" {
	//	if IsEncoding(data, encoding) {
	//		requests.DecompressBody(&data, encoding)
	//	}
	// }

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

func NewCookieJar(baseURL, cookies string) (jar http.CookieJar, err error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	jar, err = cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	slice := strings.Split(cookies, "; ")
	for _, cookie := range slice {
		kv := strings.Split(cookie, "=")
		if len(kv) < 1 {
			continue
		}

		k := strings.TrimSpace(kv[0])
		v := strings.Join(kv[1:], "=")
		jar.SetCookies(u, []*http.Cookie{{Name: k, Value: strings.TrimSpace(v)}})
	}

	// jar.SetCookies(u, []*http.Cookie{{Name: "xxx", Value: "xxx"}})
	return
}

func TextResponse(response *http.Response) (value string) {
	if response == nil {
		return
	}
	bin, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	// encoding := response.Header.Get("Content-Encoding")
	// if encoding != "" && response.Proto == "JA3" {
	//	if IsEncoding(bin, encoding) {
	//		requests.DecompressBody(&bin, encoding)
	//	}
	// }

	return string(bin)
}

// func Decode(data *[]byte, encoding string) {
//	if encoding != "" && data != nil {
//		if IsEncoding(*data, encoding) {
//			requests.DecompressBody(data, encoding)
//		}
//	}
// }
