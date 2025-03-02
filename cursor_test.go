package emit

import (
	"context"
	"github.com/bogdanfinn/tls-client/profiles"
	"net/http"
	"testing"
	"time"
)

func TestZed(t *testing.T) {
	session, err := NewSession(proxies, false, nil, Ja3Helper(Echo{
		RandomTLSExtension: true,
		HelloID:            profiles.Chrome_124,
	}, 30))
	if err != nil {
		t.Fatal(err)
	}

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	response, err := ClientBuilder(session).
		Proxies(proxies).
		Context(timeout).
		POST("https://llm.zed.dev/completion").
		Header("Origin", "https://llm.zed.dev").
		Header("Referer", "https://llm.zed.dev/").
		Header("User-Agent", "Zed/0.175.6 (macos; x86_64)").
		Header("authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpYXQiOjE3NDA5MzMxNzUsImV4cCI6MTc0MDkzNjc3NSwianRpIjoiNjVhMzdhMTAtZjc1Mi00M2IwLTk2YWEtMjdhZjFkZDgyMDhhIiwidXNlcklkIjoyNzYwMzksInN5c3RlbUlkIjoiNGI1Y2EzYzItZjlhOS00ZDllLTg3YTQtOTRhYmZlODMyZGFiIiwibWV0cmljc0lkIjoiNGM1MjMwZjEtZmY3NS00NjQzLTgyYTUtNTQ2NmM5MTlmMzUyIiwiZ2l0aHViVXNlckxvZ2luIjoiYmluY29vbyIsImlzU3RhZmYiOmZhbHNlLCJoYXNMbG1DbG9zZWRCZXRhRmVhdHVyZUZsYWciOmZhbHNlLCJoYXNQcmVkaWN0RWRpdHNGZWF0dXJlRmxhZyI6dHJ1ZSwiaGFzTGxtU3Vic2NyaXB0aW9uIjpmYWxzZSwibWF4TW9udGhseVNwZW5kSW5DZW50cyI6MTAwMCwiY3VzdG9tTGxtTW9udGhseUFsbG93YW5jZUluQ2VudHMiOm51bGwsInBsYW4iOiJGcmVlIn0.uqgbMLi3Yc_su-rX8F6GMbiZ0SNiIKrgbm05LpZq0OI").
		JSONHeader().
		Bytes([]byte(`{
  "provider": "anthropic",
  "model": "claude-3-7-sonnet-latest",
  "provider_request": {
    "model": "claude-3-7-sonnet-latest",
    "max_tokens": 8192,
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "\n\n\n你好\n"
          },
          {
            "type": "text",
            "text": "..."
}
]
}
],
"system": "",
"temperature": 1.0
}
}`)).
		DoS(http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
}
