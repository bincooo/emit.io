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
	return ist(response, "Text", "text/plain")
}

func IsHTML(response *http.Response) error {
	return ist(response, "Text", "text/html")
}

func IsSTREAM(response *http.Response) error {
	return ist(response, "Stream", "text/event-stream", "application/stream")
}

func IsPROTO(response *http.Response) error {
	return ist(response, "Proto", "application/connect+proto")
}

func Status(status int) func(response *http.Response) error {
	return func(response *http.Response) error {
		if response == nil {
			return Error{-1, "Status", "", errors.New("response is nil")}
		}
		if response.StatusCode != status {
			msg := ""
			if isJ(response.Header) {
				msg = TextResponse(response)
			}
			_ = response.Body.Close()
			return Error{response.StatusCode, "Status", msg, errors.New(response.Status)}
		}
		return nil
	}
}

func ist(response *http.Response, bus string, ts ...string) error {
	if response == nil {
		return Error{-1, bus, "", errors.New("response is nil")}
	}
	h := response.Header
	for _, t := range ts {
		if strings.Contains(h.Get("Content-Type"), t) {
			return nil
		}
	}
	msg := ""
	if isJ(response.Header) {
		msg = TextResponse(response)
	}
	_ = response.Body.Close()
	return Error{-1, bus, msg, fmt.Errorf("response is not [ %s ]", ts)}
}

func isJ(header http.Header) bool {
	if header == nil {
		return false
	}
	return strings.Contains(header.Get("Content-Type"), "application/json")
}
