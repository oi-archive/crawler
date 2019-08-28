package main

import (
	"github.com/oi-archive/crawler/plugin/public"
	"github.com/oi-archive/crawler/plugin/syzoj"
	"log"
)

const homePath = "loj/"

func Name() string {
	return "LibreOJ"
}

func Start(logg *log.Logger) error {
	return syzoj.Start(logg, homePath, Name(), "https://loj.ac")
}

/* 执行一次题库爬取
 * limit: 一次最多爬取题目数
 */
func Update(limit int) (public.FileList, error) {
	return syzoj.Update(limit)
}

func Stop() {
	syzoj.Stop()
}
