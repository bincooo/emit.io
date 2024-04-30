package emits

import (
	"context"
	"github.com/bincooo/gio.emits/common"
	"net/http"
	"testing"
	"time"
)

const (
	proxies   = "http://127.0.0.1:7890"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
)

func TestGio1(t *testing.T) {
	model := "claude-3-haiku-20240307"
	query := "hi ~"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hash := SessionHash()
	obj := map[string]interface{}{
		"event_data":   nil,
		"fn_index":     41,
		"trigger_id":   93,
		"session_hash": hash,
		"data": []interface{}{
			nil,
			model,
			query,
			nil,
		},
	}

	response, err := common.ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		POST("https://chat.lmsys.org/queue/join").
		Header("Origin", "https://chat.lmsys.org").
		Header("Referer", "https://chat.lmsys.org/").
		Header("User-Agent", userAgent).
		JHeader().
		Body(obj).
		DoWith(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	object, err := common.ToMap(response)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(object)
	cookie := common.GetCookies(response)

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err = common.ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		GET("https://chat.lmsys.org/queue/data").
		Query("session_hash", hash).
		Header("Referer", "https://chat.lmsys.org/").
		Header("User-Agent", userAgent).
		Header("Cookie", cookie).
		DoWith(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	e, err := New(ctx, response)
	if err != nil {
		t.Fatal(err)
	}

	next := false
	cookie = common.MergeCookies(cookie, common.GetCookies(response))

	e.Event("*", func(j JoinCompleted) interface{} {
		t.Log(string(j.InitialBytes))
		return nil
	})
	e.Event("process_completed", func(j JoinCompleted) interface{} {
		if j.Success {
			obj["fn_index"] = 42
			obj["data"] = []interface{}{
				nil,
				0.7,
				1,
				1024,
			}

			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			response, err = common.ClientBuilder().
				Proxies(proxies).
				Context(ctx).
				POST("https://chat.lmsys.org/queue/join").
				Header("Origin", "https://chat.lmsys.org").
				Header("Referer", "https://chat.lmsys.org/").
				Header("User-Agent", userAgent).
				Header("Cookie", cookie).
				JHeader().
				Body(obj).
				DoWith(http.StatusOK)
			if err != nil {
				t.Fatal(err)
			}

			object, err = common.ToMap(response)
			if err != nil {
				t.Fatal(err)
			}

			cookie = common.MergeCookies(cookie, common.GetCookies(response))
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

	response, err = common.ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		GET("https://chat.lmsys.org/queue/data").
		Query("session_hash", hash).
		Header("Referer", "https://chat.lmsys.org/").
		Header("User-Agent", userAgent).
		Header("Cookie", cookie).
		DoWith(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	e, err = New(ctx, response)
	if err != nil {
		t.Fatal(err)
	}

	e.Event("*", func(j JoinCompleted) interface{} {
		t.Log(string(j.InitialBytes))
		return nil
	})

	if err = e.Do(); err != nil {
		t.Fatal(err)
	}
}

func TestGio2(t *testing.T) {
	p := "1girl"
	n := ""
	conn, err := common.SocketBuilder().
		Proxies(proxies).
		URL("wss://tonyassi-text-to-image-sdxl.hf.space/queue/join").
		Header("User-Agent", userAgent).
		DoWith(http.StatusSwitchingProtocols)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	hash := SessionHash()
	e, err := New(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}

	e.Event("*", func(j JoinCompleted) interface{} {
		t.Log(string(j.InitialBytes))
		return nil
	})

	e.Event("send_hash", func(j JoinCompleted) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"session_hash": hash,
		}
	})

	e.Event("send_data", func(j JoinCompleted) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"event_data":   nil,
			"session_hash": hash,
			"data": []interface{}{
				p, n, "1024x1024",
			},
		}
	})

	e.Event("process_completed", func(j JoinCompleted) interface{} {
		if j.Success {
			t.Log("success.")
		}
		return nil
	})

	if err = e.Do(); err != nil {
		t.Fatal(err)
	}
}
