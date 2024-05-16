package emits

import "fmt"

type Error struct {
	Code int
	Bus  string
	Err  error
}

func (err Error) Error() string {
	bus := err.Bus
	if bus == "" {
		bus = "common"
	}
	return fmt.Sprintf("<%s: %d> %v", bus, err.Code, err.Err)
}
