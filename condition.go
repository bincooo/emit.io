package emit

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func IsJSON(response *http.Response) error {
	return ist(response, "Json", "application/json")
}

func IsTEXT(response *http.Response) error {
	return ist(response, "Text", "text/html")
}

func IsSTREAM(response *http.Response) error {
	return ist(response, "Stream", "text/event-stream", "application/stream")
}

func Status(status int) func(response *http.Response) error {
	return func(response *http.Response) error {
		if response == nil {
			return Error{-1, "Status", errors.New("response is nil")}
		}
		if response.StatusCode != status {
			return Error{response.StatusCode, "Status", errors.New(response.Status)}
		}
		return nil
	}
}

func ist(response *http.Response, bus string, ts ...string) error {
	if response == nil {
		return Error{-1, bus, errors.New("response is nil")}
	}
	h := response.Header
	for _, t := range ts {
		if strings.Contains(h.Get("content-type"), t) {
			return nil
		}
	}
	return Error{-1, bus, fmt.Errorf("response is not [ %s ]", ts)}
}
