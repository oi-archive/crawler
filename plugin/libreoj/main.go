package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type lojExportProblem struct {
	Success bool
	Obj     struct {
		Title              string
		Description        string
		InputFormat        string `json:"input_format"`
		OutputFormat       string `json:"output_format"`
		Example            string
		LimitAndHint       string `json:"limit_and_hint"`
		TimeLimit          int    `json:"time_limit"`
		MemoryLimit        int    `json:"memory_limit"`
		HaveAdditionalFile bool   `json:"have_additional_file"`
		FileIO             bool   `json:"file_io"`
		Type               string
		Tags               []string
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

const homePath = "loj/"

type problem struct {
	Time        int    `json:"time"`
	Memory      int    `json:"memory"`
	Title       string `json:"title"`
	Judge       string `json:"judge"`
	Url         string `json:"url"`
	description string
}

type problemListItem struct {
	Title string `json:"title"`
	Pid   string `json:"pid"`
	data  *problem
}

type problemList []problemListItem

var fileList map[string][]byte

func writeProblemList(list problemList) error {
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	fileList[homePath+"problemlist.json"] = b
	return nil
}

func writeMainJson(path string, p *problemListItem) error {
	b, err := json.Marshal(p.data)
	if err != nil {
		return err
	}
	fileList[path] = b
	return nil
}

func Name() string {
	return "LibreOJ"
}

func Start() error {
	log.Println("LibreOJ crawler started")
	return nil
}

func safeGet(url string) (res *http.Response, err error) {
	for i := 1; i <= 3; i++ {
		res, err = http.Get(url)
		if err != nil {
			time.Sleep(time.Millisecond * 50)
			continue
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf("get %s error,status code = %d", url, res.StatusCode)
			time.Sleep(time.Millisecond * 50)
			continue
		}
		return res, nil
	}
	return nil, err
}

func getDocument(url string) (*goquery.Document, error) {
	res, err := safeGet(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

/* 执行一次题库爬取
 * limit: 一次最多爬取题目数
 */
func Update(limit int) (map[string][]byte, error) {
	fileList = make(map[string][]byte)
	problemPage, err := getDocument("https://loj.ac/problems")
	if err != nil {
		return nil, err
	}
	list := problemPage.Find(".ui.pagination.menu")
	maxPage := 0
	if list.Size() > 0 {
		for i := range list.Eq(0).Nodes {
			//tt:=list.Eq(i).Text()
			for _, j := range strings.Split(list.Eq(i).Text(), "\n") {
				t, err := strconv.Atoi(j)
				if err != nil {
					continue
				}
				if t > maxPage {
					maxPage = t
				}
			}
		}
	}
	if maxPage <= 0 || maxPage >= 500 {
		return nil, fmt.Errorf("maxPage value error: %d", maxPage)
	}
	maxPage = 1
	newPList := make([]problemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		problemListPage, err := getDocument(fmt.Sprintf("https://loj.ac/problems?page=%d", i))
		if err != nil {
			return nil, err
		}
		list := problemListPage.Find(`[style="vertical-align: middle; "]`)
		for j := range list.Nodes {
			p := problemListItem{Title: strings.Replace(list.Eq(j).Text(), "\n", "", -1)}
			node := list.Nodes[j]
			for _, k := range node.Attr {
				if k.Key == "href" {
					p.Pid = strings.Split(k.Val, "/")[2] //TODO: 异常未处理
					break
				}
			}
			newPList = append(newPList, p)
		}
	}
	for k := range newPList {
		i := &newPList[k]
		log.Println("start getting problem ", i.Pid)
		i.data = nil
		res, err := safeGet(fmt.Sprintf("https://loj.ac/problem/%s/export", i.Pid))
		if err != nil {
			continue
		}
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			continue
		}
		res.Body.Close()
		data := &lojExportProblem{}
		err = json.Unmarshal(b, data)
		if err != nil {
			continue
		}
		if !data.Success {
			continue
		}
		i.data = &problem{}
		i.data.Title = i.Title
		i.data.Time = data.Obj.TimeLimit
		i.data.Memory = data.Obj.MemoryLimit
		i.data.Url = "https://loj.ac/problem/" + i.Pid
		switch data.Obj.Type {
		case "traditional":
			i.data.Judge = "传统"
		case "submit-answer":
			i.data.Judge = "提交答案"
		case "interaction":
			i.data.Judge = "交互"
		}
		for _, k := range data.Obj.Tags {
			if k == "Special Judge" {
				i.data.Judge += " Special Judge"
				break
			}
		}
		i.data.description = fmt.Sprintf(
			`
# 题目描述

%s

# 输入格式

%s

# 输出格式

%s

# 样例

%s

# 数据范围与提示

%s

`, data.Obj.Description, data.Obj.InputFormat, data.Obj.OutputFormat, data.Obj.Example, data.Obj.LimitAndHint)
	}
	err = writeProblemList(newPList)
	if err != nil {
		return nil, err
	}
	for _, i := range newPList {
		if i.data == nil {
			continue
		}
		nowPath := homePath + i.Pid + "/"
		err = writeMainJson(nowPath+"main.json", &i)
		if err != nil {
			log.Println(err)
			continue
		}
		fileList[nowPath+"description.md"] = []byte(i.data.description)
	}
	return fileList, nil
}

func Stop() {
	log.Println("LibreOJ crawler stopped")
}
