package websocket

import "github.com/eris-ltd/new-thelonious/thelutil"

type Message struct {
	Call  string        `json:"call"`
	Args  []interface{} `json:"args"`
	Id    int           `json:"_id"`
	Data  interface{}   `json:"data"`
	Event string        `json:"_event"`
}

func (self *Message) Arguments() *thelutil.Value {
	return thelutil.NewValue(self.Args)
}
