package emit

import (
	"context"
	"github.com/bogdanfinn/tls-client/profiles"
	"net/http"
	"testing"
)

func TestJa3(t *testing.T) {
	session, err := NewSession(proxies, nil, Ja3Helper(
		Echo{true, profiles.CloudflareCustom},
		120,
	))
	if err != nil {
		t.Fatal(err)
	}

	response, err := ClientBuilder(session).
		Context(context.Background()).
		GET("https://tls.browserleaks.com/json").
		Ja3().
		Header("user-agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	defer response.Body.Close()
	obj, err := ToMap(response)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ja3: %s", obj["ja3_text"])
	t.Logf("ja3_hash: %s", obj["ja3_hash"])
	t.Logf("user_agent: %s", obj["user_agent"])

	//    http_test.go:37: ja3: 771,49195-49199-49196-49200-52393-52392-49161-49171-49162-49172-156-157-47-53-49170-10-4865-4866-4867,0-5-10-11-13-65281-23-18-43-51,29-23-24-25,0
	//    http_test.go:38: ja3_hash: 1be8360b66649edee1de25f81d98ec27
	//    http_test.go:39: user_agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0
}

func TestHttp(t *testing.T) {
	session, err := NewSession(proxies, WarpI("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}

	response, err := ClientBuilder(session).
		Context(context.Background()).
		GET("https://tls.browserleaks.com/json").
		Header("user-agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	defer response.Body.Close()
	obj, err := ToMap(response)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ja3: %s", obj["ja3_text"])
	t.Logf("ja3_hash: %s", obj["ja3_hash"])
	t.Logf("user_agent: %s", obj["user_agent"])
}

func TestEncoding(t *testing.T) {
	session, err := NewSession(proxies, WarpI("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}

	response, err := ClientBuilder(session).
		Context(context.Background()).
		GET("https://claude.ai/_next/static/css/42239112c73b3fbe.css").
		Header("user-agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	defer response.Body.Close()
	obj, err := ToMap(response)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ja3: %s", obj["ja3_text"])
	t.Logf("ja3_hash: %s", obj["ja3_hash"])
	t.Logf("user_agent: %s", obj["user_agent"])
}
