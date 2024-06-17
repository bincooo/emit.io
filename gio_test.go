package emit

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	proxies     = "http://127.0.0.1:7890"
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0"
	baseCookies = "_ga=GA1.1.1320014795.1715641484; _ga_K6D24EE9ED=GS1.1.1717132441.24.0.1717132441.0.0.0; _ga_R1FN4KJKJH=GS1.1.1717132441.38.0.1717132441.0.0.0"
)

func TestRandIP(t *testing.T) {
	t.Log(RandIP())
}

func TestClaude3Haiku20240307(t *testing.T) {
	model := "claude-3-haiku-20240307"
	query := "hi ~"

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	hash := GioHash()
	obj := map[string]interface{}{
		"fn_index":     44,
		"trigger_id":   95,
		"session_hash": hash,
		"data": []interface{}{
			nil,
			model,
			query,
			nil,
		},
	}

	cookies := fetchCookies(ctx, proxies)
	response, err := ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		Option(ConnectOption{
			IdleConnTimeout: 10 * time.Second,
		}).
		POST("https://chat.lmsys.org/queue/join").
		Header("Origin", "https://chat.lmsys.org").
		Header("Referer", "https://chat.lmsys.org/").
		Header("User-Agent", userAgent).
		Header("cookie", cookies).
		JHeader().
		Body(obj).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	object, err := ToMap(response)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(object)
	cookie := GetCookies(response)

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err = ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		GET("https://chat.lmsys.org/queue/data").
		Query("session_hash", hash).
		Header("Referer", "https://chat.lmsys.org/").
		Header("User-Agent", userAgent).
		Header("Cookie", cookie).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	e, err := NewGio(ctx, response)
	if err != nil {
		t.Fatal(err)
	}

	next := false
	cookie = MergeCookies(cookie, GetCookies(response))

	e.Event("*", func(j JoinEvent) interface{} {
		t.Log(string(j.InitialBytes))
		return nil
	})
	e.Event("process_completed", func(j JoinEvent) interface{} {
		if j.Success {
			obj["fn_index"] = 45
			obj["data"] = []interface{}{
				nil,
				0.7,
				1,
				1024,
			}

			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			response, err = ClientBuilder().
				Proxies(proxies).
				Context(ctx).
				POST("https://chat.lmsys.org/queue/join").
				Header("Origin", "https://chat.lmsys.org").
				Header("Referer", "https://chat.lmsys.org/").
				Header("User-Agent", userAgent).
				Header("Cookie", cookie).
				JHeader().
				Body(obj).
				DoS(http.StatusOK)
			if err != nil {
				t.Fatal(err)
			}

			object, err = ToMap(response)
			if err != nil {
				t.Fatal(err)
			}

			cookie = MergeCookies(cookie, GetCookies(response))
			t.Log(object)
			next = true
		}
		return nil
	})

	if err = e.Do(); err != nil || !next {
		if err != nil {
			t.Fatal(err)
		}
		t.Fatal("break off")
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err = ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		GET("https://chat.lmsys.org/queue/data").
		Query("session_hash", hash).
		Header("Referer", "https://chat.lmsys.org/").
		Header("User-Agent", userAgent).
		Header("Cookie", cookie).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	e, err = NewGio(ctx, response)
	if err != nil {
		t.Fatal(err)
	}

	e.Event("*", func(j JoinEvent) interface{} {
		t.Log(string(j.InitialBytes))
		return nil
	})

	if err = e.Do(); err != nil {
		t.Fatal(err)
	}
}

func TestGioSDXL(t *testing.T) {
	p := "1girl"
	n := ""
	conn, err := SocketBuilder().
		Proxies(proxies).
		URL("wss://tonyassi-text-to-image-sdxl.hf.space/queue/join").
		Header("User-Agent", userAgent).
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	hash := GioHash()
	e, err := NewGio(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}

	e.Event("*", func(j JoinEvent) interface{} {
		t.Log(string(j.InitialBytes))
		return nil
	})

	e.Event("send_hash", func(j JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"session_hash": hash,
		}
	})

	e.Event("send_data", func(j JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"event_data":   nil,
			"session_hash": hash,
			"data": []interface{}{
				p, n, "1024x1024",
			},
		}
	})

	e.Event("process_completed", func(j JoinEvent) interface{} {
		if j.Success {
			t.Log("success.")
		}
		return nil
	})

	if err = e.Do(); err != nil {
		t.Fatal(err)
	}
}

func fetchCookies(ctx context.Context, proxies string) (cookies string) {
	retry := 3
label:
	if retry <= 0 {
		return
	}
	retry--
	response, err := ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET("https://chat.lmsys.org/info").
		Header("Accept-Language", "en-US,en;q=0.9").
		//Header("Origin", "https://chat.lmsys.org").
		Header("Host", "chat.lmsys.org").
		Header("Referer", "https://chat.lmsys.org/").
		Header("cookie", baseCookies).
		Header("User-Agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		fmt.Println(err)
		goto label
	}

	cookie := GetCookie(response, "SERVERID")
	if cookie == "" {
		goto label
	}

	co := strings.Split(cookie, "|")
	if len(co) < 2 {
		goto label
	}

	if len(co[0]) < 1 || co[0][0] != 'S' {
		goto label
	}

	if co[0] == "S0" {
		goto label
	}
	cookies = fmt.Sprintf("SERVERID=%s", cookie)
	cookies = MergeCookies(baseCookies, cookies)
	return
}

func TestAPI(t *testing.T) {
	response, err := ClientBuilder().
		POST("http://127.0.0.1:8000/chat").
		JHeader().
		Body(map[string]interface{}{
			"previousMessages": make([]string, 0),
			"message":          "帮我打开计算器",
		}).DoC(Status(http.StatusOK), IsSTREAM)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if eof {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for {
		if !scanner.Scan() {
			break
		}

		data := scanner.Text()
		if len(data) < 6 || data[:6] != "data: " {
			continue
		}

		data = data[6:]
		if data == "[DONE]" {
			break
		}

		t.Log(data)
	}
}
