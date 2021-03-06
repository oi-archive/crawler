package main

import (
	. "crawler/plugin/public"
	"fmt"
	"log"
	"strconv"
	"strings"
)

const homePath = "cogs/"

var logger *log.Logger

var fileList map[string][]byte

var oldPList map[string]bool
var lastPoint string

func Name() string {
	return "COGS"
}

func Start(logg *log.Logger) error {
	logger = logg
	oldPList = make(map[string]bool)
	err := InitPList(oldPList, homePath)
	if err != nil {
		return err
	}
	lastPoint = ""
	logger.Println("COGS crawler started")
	return nil
}

func Update(limit int) (FileList, error) {
	logger.Println("Updating COGS")
	fileList = make(FileList)
	problemPage, err := GetDocument(nil, "http://cogs.pro:8080/cogs/problem/index.php")
	if err != nil {
		return nil, err
	}
	errParsingProblemList := fmt.Errorf("解析题目列表时产生错误")
	//errParsingProblem := fmt.Errorf("解析题面时产生错误")
	list := problemPage.Find(`#body > div > div > ul`)
	if len(list.Nodes) == 0 {
		return nil, errParsingProblemList
	}
	maxPage := 0
	for i := list.Nodes[0].FirstChild; i != nil; i = i.NextSibling {
		po := i.FirstChild
		if po == nil {
			continue
		}
		po = po.FirstChild
		if po == nil {
			continue
		}
		t, err := strconv.Atoi(po.Data)
		if err == nil && t > maxPage {
			maxPage = t
		}
	}
	if maxPage <= 0 || maxPage >= 500 {
		return nil, fmt.Errorf("maxPage error: %d", maxPage)
	}
	newPList := make([]ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		problemListPage, err := GetDocument(nil, fmt.Sprintf("http://cogs.pro:8080/cogs/problem/index.php?page=%d", i))
		if err != nil {
			return nil, err
		}
		cnt := 0
		for j := 1; j <= 30; j++ { // 可能 runtime error
			p := ProblemListItem{}
			p.Data = &Problem{}
			po := problemListPage.Find(fmt.Sprintf(`#problist > tbody > tr:nth-child(%d) > td:nth-child(1)`, j))
			if len(po.Nodes) == 0 {
				break
			}
			p.Pid = po.Nodes[0].FirstChild.Data
			po2 := problemListPage.Find(fmt.Sprintf(`#problist > tbody > tr:nth-child(%d) > td:nth-child(2) > b > a`, j)).Nodes[0]
			for _, k := range po2.Attr {
				if k.Key == "href" {
					p.Data.Url = "http://cogs.pro:8080/cogs/problem/" + k.Val
				}
			}
			if p.Data.Url == "" {
				return nil, errParsingProblemList
			}
			p.Title = po2.FirstChild.Data
			p.Data.Title = p.Title
			t, err := strconv.ParseFloat(strings.Split(problemListPage.Find(fmt.Sprintf(`#problist > tbody > tr:nth-child(%d) > td:nth-child(4)`, j)).Nodes[0].FirstChild.Data, " ")[0], 64)
			if err != nil {
				return nil, err
			}
			p.Data.Time = int(t) * 1000
			t, err = strconv.ParseFloat(strings.Split(problemListPage.Find(fmt.Sprintf(`#problist > tbody > tr:nth-child(%d) > td:nth-child(5)`, j)).Nodes[0].FirstChild.Data, " ")[0], 64)
			if err != nil {
				return nil, err
			}
			p.Data.Memory = int(t)
			newPList = append(newPList, p)
			cnt++
		}
		if cnt == 0 {
			return nil, errParsingProblemList
		}
	}
	lastPoint = DownloadProblems(newPList, oldPList, limit, lastPoint, func(p *ProblemListItem) error {
		logger.Println("开始抓取题目 ", p.Pid)
		page, err := GetDocument(nil, p.Data.Url)
		if err != nil {
			return fmt.Errorf("下载题面失败: %v", err)
		}
		html := NodeChildren2html(page.Find(`#probdetail > dl`).Nodes[0])
		html += "\n"
		html = "# 题目描述\n\n" + html
		p.Data.Description = html
		p.Data.DescriptionType = "html"
		p.Data.Judge = page.Find(`#leftbar > table:nth-child(1) > tbody > tr:nth-child(6) > td > span.pull-right > span`).Nodes[0].FirstChild.Data
		return nil
	})
	err = WriteFiles(newPList, fileList, homePath)
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
	logger.Println("COGS crawler stopped")
}
