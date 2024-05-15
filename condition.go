package emits

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func IsJSON(response *http.Response) error {
	return ist(response, "application/json")
}

func IsTEXT(response *http.Response) error {
	return ist(response, "text/html")
}

func IsSTREAM(response *http.Response) error {
	return ist(response, "text/event-stream")
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

func ist(response *http.Response, t string) error {
	if response == nil {
		return errors.New("response is nil")
	}
	h := response.Header
	if strings.Contains(h.Get("content-type"), t) {
		return nil
	}
	return fmt.Errorf("response is not '%s'", t)
}
