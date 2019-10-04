package main

import (
	"crawler/plugin/public"
	"crawler/plugin/syzoj"
	"log"
)

const homePath = "guoj/"

var c *syzoj.SYZOJ

func Name() string {
	return "GuOJ"
}

func Start(logg *log.Logger) error {
	c = &syzoj.SYZOJ{}
	return c.Start(logg, homePath, Name(), "https://guoj.icu")
}

/* 执行一次题库爬取
 * limit: 一次最多爬取题目数
 */
func Update(limit int) (public.FileList, error) {
	return c.Update(limit)
}

func Stop() {
	c.Stop()
}
