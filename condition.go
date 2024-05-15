package emits

import (
	"errors"
	"net/http"
	"strings"
)

func IsJSON(response *http.Response) error {
	if response == nil {
		return errors.New("response is nil")
	}
	h := response.Header
	t := h.Get("content-type")
	if strings.Contains(t, "application/json") {
		return nil
	}
	return errors.New("response is not 'application/json'")
}

func IsTEXT(response *http.Response) error {
	if response == nil {
		return errors.New("response is nil")
	}
	h := response.Header
	t := h.Get("content-type")
	if strings.Contains(t, "text/html") {
		return nil
	}
	return errors.New("response is not 'text/html'")
}

func Status(status int) func(response *http.Response) error {
	return func(response *http.Response) error {
		if response == nil {
			return errors.New("response is nil")
		}
		if response.StatusCode != status {
			return errors.New(response.Status)
		}
		return nil
	}
}
