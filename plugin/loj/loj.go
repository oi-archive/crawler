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
	err := c.Start(&rpc.Info{Id: "loj", Name: "LibreOJ"}, "https://loj.ac")
	if err != nil {
		log.Panicln(err)
	}
	err = c.Update(200)
	if err != nil {
		log.Panicln(err)
	}
	c.Stop()
}
