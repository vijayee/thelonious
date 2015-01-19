package rpc

import "github.com/eris-ltd/new-thelonious/monkutil"

type Message struct {
	Call string        `json:"call"`
	Args []interface{} `json:"args"`
	Id   int           `json:"_id"`
	Data interface{}   `json:"data"`
}

func (self *Message) Arguments() *monkutil.Value {
	return monkutil.NewValue(self.Args)
}
