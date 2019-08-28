package main

import (
	"fmt"
	"github.com/oi-archive/crawler/plugin/public"
	"log"
	"regexp"
	"strconv"
	"strings"
)

const homePath = "uoj/"

var logger *log.Logger

var fileList map[string][]byte

var oldPList map[string]bool
var lastPoint string

func Name() string {
	return "UniversalOJ"
}

func Start(logg *log.Logger) error {
	logger = logg
	oldPList = make(map[string]bool)
	err := public.InitPList(oldPList, homePath)
	if err != nil {
		return err
	}
	lastPoint = ""
	logger.Println("UniversalOJ crawler started")
	return nil
}

func Update(limit int) (public.FileList, error) {
	fileList = make(public.FileList)
	problemPage, err := public.GetDocument(nil, "http://uoj.ac/problems")
	if err != nil {
		return nil, err
	}
	errParsingProblemList := fmt.Errorf("解析 UniversalOJ 题目列表时产生错误")
	errParsingProblem := fmt.Errorf("解析题面时产生错误")
	list := problemPage.Find(`body > div > div.uoj-content > div.row > div.col-sm-4.col-sm-pull-4 > div > ul`)
	if len(list.Nodes) == 0 {
		return nil, errParsingProblemList
	}
	maxPage := 0
	for i := list.Nodes[0].FirstChild; i != nil; i = i.NextSibling {
		t := i.FirstChild
		if t != nil {
			t = t.FirstChild
			if t != nil {
				x, err := strconv.Atoi(t.Data)
				if err == nil && x > maxPage {
					maxPage = x
				}
			}
		}
	}
	if maxPage <= 0 || maxPage >= 500 {
		return nil, fmt.Errorf("maxPage error: %d", maxPage)
	}
	newPList := make([]public.ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		problemListPage, err := public.GetDocument(nil, fmt.Sprintf("http://uoj.ac/problems?page=%d", i))
		if err != nil {
			return nil, err
		}
		table := problemListPage.Find(`body > div > div.uoj-content > div.table-responsive > table > tbody`)
		if len(table.Nodes) == 0 {
			return nil, errParsingProblemList
		}
		for j := table.Nodes[0].FirstChild; j != nil; j = j.NextSibling {
			p := public.ProblemListItem{}
			po := j.FirstChild
			if po == nil || po.FirstChild == nil {
				return nil, errParsingProblemList
			}
			p.Pid = strings.Replace(po.FirstChild.Data, "#", "", -1)
			po = po.NextSibling
			if po == nil || po.FirstChild == nil || po.FirstChild.FirstChild == nil {
				return nil, errParsingProblemList
			}
			p.Title = po.FirstChild.FirstChild.Data
			newPList = append(newPList, p)
		}
	}
	lastPoint = public.DownloadProblems(newPList, oldPList, limit, lastPoint, func(p *public.ProblemListItem) error {
		logger.Println("开始抓取题目 ", p.Pid)
		p.Data = nil
		page, err := public.GetDocument(nil, `http://uoj.ac/problem/`+p.Pid)
		if err != nil {
			return fmt.Errorf("下载题面失败: %v", err)
		}
		x := page.Find(`#tab-statement > article`)
		if len(x.Nodes) == 0 {
			return errParsingProblem
		}
		p.Data = &public.Problem{}
		html := public.Node2html(x.Nodes[0])
		html = strings.Replace(html, `<article class="top-buffer-md">`, "", -1)
		html = strings.Replace(html, `</article>`, "", -1)
		rule := regexp.MustCompile(`<h3>.+?</h3>`)
		html = rule.ReplaceAllStringFunc(html, func(x string) string {
			return "# " + x[4:len(x)-5] + "\n\n"
		})
		rule = regexp.MustCompile(`时间限制(?:</strong>)*：(?:</strong>)*\$(.+?)\\texttt{s}\$`)
		match := rule.FindStringSubmatch(html)
		logger.Println(len(match))
		if len(match) > 0 {
			t := match[1]
			t = strings.Trim(t, " ")
			time, err := strconv.Atoi(t)
			if err == nil {
				p.Data.Time = time * 1000
			}
		}
		rule = regexp.MustCompile(`(?:空间|内存)限制(?:</strong>)*：(?:</strong>)*\$(.+?)\\texttt{([MG])B}\$`)
		match = rule.FindStringSubmatch(html)
		logger.Println(len(match))
		if len(match) > 0 {
			t := match[1]
			t = strings.Trim(t, " ")
			memory, err := strconv.Atoi(t)
			if err == nil {
				if match[2] == "M" {
					p.Data.Memory = memory
				} else if match[2] == "G" {
					p.Data.Memory = memory * 1024
				}
			}
		}
		p.Data.Description = "# 题目描述\n\n" + html
		p.Data.Title = p.Title
		p.Data.Url = "http://uoj.ac/problem/" + p.Pid
		p.Data.DescriptionType = "html"
		if p.Data.Time == 0 {
			p.Data.Judge = "提交答案"
		} else {
			p.Data.Judge = "传统或交互"
		}
		return nil
	})
	err = public.WriteFiles(newPList, fileList, homePath)
	if err != nil {
		return nil, err
	}
	oldPList = make(map[string]bool)
	for _, i := range newPList {
		oldPList[i.Pid] = true
	}
	return fileList, nil
}

func Stop() {
	logger.Println("UniversalOJ crawler stopped")
}
