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
	err := c.Start(&rpc.Info{Id: "seuoj", Name: "seuOJ"}, "https://oj.seucpc.club")
	if err != nil {
		log.Panicln(err)
	}
	err = c.Update(50)
	if err != nil {
		log.Panicln(err)
	}
	c.Stop()
}
