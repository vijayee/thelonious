package websocket

import "github.com/eris-ltd/new-thelonious/ethutil"

type Message struct {
	Call  string        `json:"call"`
	Args  []interface{} `json:"args"`
	Id    int           `json:"_id"`
	Data  interface{}   `json:"data"`
	Event string        `json:"_event"`
}

func (self *Message) Arguments() *ethutil.Value {
	return ethutil.NewValue(self.Args)
}
