package main

import (
	"crawler/plugin/public"
	"crawler/plugin/syzoj"
	"log"
)

const homePath = "seuoj/"

func Name() string {
	return "seuOJ"
}

var c *syzoj.SYZOJ

func Start(logg *log.Logger) error {
	c = &syzoj.SYZOJ{}
	return c.Start(logg, homePath, Name(), "https://oj.seucpc.club")
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