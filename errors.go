package emit

import "fmt"

type Error struct {
	Code int
	Bus  string
	Msg  string
	Err  error
}

func (err Error) Error() (result string) {
	bus := err.Bus
	if bus == "" {
		bus = "common"
	}
	result = fmt.Sprintf("<%s: %d> %v", bus, err.Code, err.Err)
	if err.Msg != "" {
		result = fmt.Sprintf("%s & %s", result, err.Msg)
	}
	return
}
