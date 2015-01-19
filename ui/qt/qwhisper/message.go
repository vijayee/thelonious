package qwhisper

import (
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/monkutil"
	"github.com/eris-ltd/new-thelonious/whisper"
)

type Message struct {
	ref     *whisper.Message
	Flags   int32  `json:"flags"`
	Payload string `json:"payload"`
	From    string `json:"from"`
}

func ToQMessage(msg *whisper.Message) *Message {
	return &Message{
		ref:     msg,
		Flags:   int32(msg.Flags),
		Payload: "0x" + monkutil.Bytes2Hex(msg.Payload),
		From:    "0x" + monkutil.Bytes2Hex(crypto.FromECDSAPub(msg.Recover())),
	}
}
