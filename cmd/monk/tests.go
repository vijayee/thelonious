package main

import (
	"github.com/eris-ltd/thelonious/monk"
	"time"
)

func RunTest(m *monk.MonkModule, test string) {
	switch test {
	case "load":
		TestRunLoad(m)
	}
}

func TestRunLoad(m *monk.MonkModule) {
	m.Init()
	m.Start()
	go func() {
		tick := time.Tick(1000 * time.Millisecond)
		addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
		amount := "567890"
		for _ = range tick {
			m.Tx(addr, amount)
		}
	}()
	m.WaitForShutdown()
}
