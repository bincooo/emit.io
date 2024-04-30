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

type JoinCompleted struct {
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

type Emits struct {
	response *http.Response
	conn     *websocket.Conn
	ctx      context.Context
	em       map[string]func(j JoinCompleted) interface{}
}

func New(ctx context.Context, coupler interface{}) (e *Emits, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	e = &Emits{
		ctx: ctx,
		em:  map[string]func(j JoinCompleted) interface{}{
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

func (e *Emits) Event(eventId string, funcCall func(j JoinCompleted) interface{}) {
	e.em[eventId] = funcCall
}

func (e *Emits) Do() error {
	if e.conn == nil && e.response == nil {
		panic("'coupler' is nil")
	}

	if e.conn != nil {
		return e.doConn()
	} else {
		return e.doResponse()
	}
}

func (e *Emits) doConn() error {
	for {
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
			_, data, err := e.conn.ReadMessage()
			if err != nil {
				return err
			}

			var j JoinCompleted
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

func (e *Emits) doResponse() error {
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
			if !scanner.Scan() {
				return nil
			}

			data := scanner.Text()
			if len(data) < 6 || data[:6] != "data: " {
				continue
			}
			data = data[6:]

			var j JoinCompleted
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

func SessionHash() string {
	bin := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	binL := len(bin)
	var buf bytes.Buffer
	for x := 0; x < 10; x++ {
		ch := bin[rand.Intn(binL-1)]
		buf.WriteByte(ch)
	}

	return buf.String()
}
