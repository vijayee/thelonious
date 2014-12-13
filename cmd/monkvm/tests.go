package main

import (
	"fmt"
	"github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkutil"
	"path"
)

var (
	TestRoot = "tests"
)

func runTest(t string) {
	switch t {
	case "rlpdecode":
		TestRlpDecode()
	case "rlpdecodeencode":
		TestRlpDecodeEncode()
	default:
		fmt.Println("Unknown test")
	}
}

func TestRlpDecode() {
	env := NewVmEnv()

	code, err := monk.CompileLLL(path.Join(TestRoot, "rlpdecode.lll"), false)
	if err != nil {
		exit(err)
	}
	if len(code) > 2 && code[:2] == "0x" {
		code = code[2:]
	}

	gen := monkchain.NewBlockFromBytes(monkutil.Encode(monkchain.Genesis))
	header := gen.Header()
	rlpData := monkutil.Encode(header)
	fmt.Println("rlp data:", rlpData)

	exec(env, monkutil.Hex2Bytes(code), rlpData)

	dumpState(env.state)
}

func TestRlpDecodeEncode() {
	env := NewVmEnv()

	code, err := monk.CompileLLL(path.Join(TestRoot, "rlp-decode-encode.lll"), false)
	if err != nil {
		exit(err)
	}
	if len(code) > 2 && code[:2] == "0x" {
		code = code[2:]
	}

	gen := monkchain.NewBlockFromBytes(monkutil.Encode(monkchain.Genesis))
	header := gen.Header()
	rlpData := monkutil.Encode(header)
	fmt.Printf("rlp data: %x\n", rlpData)

	exec(env, monkutil.Hex2Bytes(code), rlpData)

	dumpState(env.state)
}
