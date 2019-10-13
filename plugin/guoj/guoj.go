package main

import (
	"crawler/plugin/syzoj"
	"crawler/rpc"
	"log"
)

var c *syzoj.SYZOJ

func Stop() {
	c.Stop()
}

func main() {
	c = &syzoj.SYZOJ{}
	err := c.Start(&rpc.Info{Id: "guoj", Name: "GuOJ"}, "https://guoj.icu")
	if err != nil {
		log.Panicln(err)
	}
	err = c.Update(50)
	if err != nil {
		log.Panicln(err)
	}
	c.Stop()
}
