package main

import (
	"github.com/eris-ltd/thelonious/monk"
)

func main() {

	m := monk.NewMonk(nil)
	m.Init()
	m.Start()

	for {
		select {}
	}

}
