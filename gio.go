package emits

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/RomiChan/websocket"
	"math/rand"
	"net/http"
)

type JoinEvent struct {
	Msg     string      `json:"msg"`
	EventId string      `json:"event_id"`
	Success bool        `json:"success"`
	Output  *joinOutput `json:"output"`

	InitialBytes []byte `json:"-"`
}

type joinOutput struct {
	Generating      bool    `json:"is_generating"`
	Duration        float64 `json:"duration"`
	AverageDuration float64 `json:"average_duration"`

	Data []interface{} `json:"data"`
}

type GioEmits struct {
	response *http.Response
	conn     *websocket.Conn
	ctx      context.Context
	em       map[string]func(j JoinEvent) interface{}
	err      error
	close    bool
}

func NewGio(ctx context.Context, coupler interface{}) (e *GioEmits, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	e = &GioEmits{
		ctx: ctx,
		em:  map[string]func(j JoinEvent) interface{}{
			//
		},
	}

	switch c := coupler.(type) {
	case *http.Response:
		e.response = c
	case *websocket.Conn:
		e.conn = c
	default:
		return nil, errors.New("'coupler' must be *http.Response or *websocket.Conn")
	}
	return
}

// 注册事件
func (e *GioEmits) Event(eventId string, funcCall func(j JoinEvent) interface{}) {
	e.em[eventId] = funcCall
}

// 异常设置，并终止事件
func (e *GioEmits) Failed(err error) {
	e.err = err
	e.Cancel()
}

// 终止事件
func (e *GioEmits) Cancel() {
	e.close = true
}

func (e *GioEmits) Do() error {
	if e.conn == nil && e.response == nil {
		panic("'coupler' is nil, please provide a valid 'coupler' value")
	}

	if e.conn != nil {
		return e.warpE(e.doConn())
	} else {
		return e.warpE(e.doResponse())
	}
}

// 异步执行
func (e *GioEmits) DoAsync() chan error {
	if e.conn == nil && e.response == nil {
		panic("'coupler' is nil, please provide a valid 'coupler' value")
	}

	err := make(chan error, 1)
	go func() {
		err <- e.Do()
	}()
	return err
}

func (e *GioEmits) warpE(err error) error {
	if e.err != nil {
		return e.err
	}
	return err
}

func (e *GioEmits) doConn() error {
	for {
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
			if e.close {
				return nil
			}

			_, data, err := e.conn.ReadMessage()
			if err != nil {
				return err
			}

			var j JoinEvent
			err = json.Unmarshal(data, &j)
			if err != nil {
				return err
			}

			var marshal []byte
			j.InitialBytes = data

			if funcCall, ok := e.em[j.Msg]; ok {
				if r := funcCall(j); r != nil {
					marshal, err = json.Marshal(r)
					if err != nil {
						return err
					}

					err = e.conn.WriteMessage(websocket.TextMessage, marshal)
					if err != nil {
						return err
					}
				}
			}

			if funcCall, ok := e.em["*"]; ok {
				if r := funcCall(j); r != nil {
					marshal, err = json.Marshal(r)
					if err != nil {
						return err
					}

					err = e.conn.WriteMessage(websocket.TextMessage, marshal)
					if err != nil {
						return err
					}
				}
			}

			if j.Success && j.Msg == "process_completed" {
				return nil
			}
		}
	}
}

func (e *GioEmits) doResponse() error {
	scanner := bufio.NewScanner(e.response.Body)
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
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
			if e.close {
				return nil
			}

			if !scanner.Scan() {
				return nil
			}

			data := scanner.Text()
			if len(data) < 6 || data[:6] != "data: " {
				continue
			}
			data = data[6:]

			var j JoinEvent
			j.InitialBytes = []byte(data)

			err := json.Unmarshal(j.InitialBytes, &j)
			if err != nil {
				return err
			}

			if funcCall, ok := e.em[j.Msg]; ok {
				funcCall(j)
			}

			if funcCall, ok := e.em["*"]; ok {
				funcCall(j)
			}

			if j.Success && j.Msg == "process_completed" {
				return nil
			}
		}
	}
}

func GioHash() string {
	bin := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	binL := len(bin)
	var buf bytes.Buffer
	for x := 0; x < 10; x++ {
		ch := bin[rand.Intn(binL-1)]
		buf.WriteByte(ch)
	}

	return buf.String()
}
