package syzoj

import (
	"encoding/json"
	"fmt"
	"github.com/oi-archive/crawler/plugin/public"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type syzojExportProblem struct {
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

type SYZOJ struct {
	homePath  string
	fullName  string
	homeUrl   string
	logger    *log.Logger
	fileList  public.FileList
	oldPList  map[string]bool
	lastPoint string
}

func (c *SYZOJ) Start(logg *log.Logger, hp string, fn string, hu string) error {
	c.homePath = hp
	c.fullName = fn
	c.homeUrl = hu
	c.logger = logg
	c.oldPList = make(map[string]bool)
	err := public.InitPList(c.oldPList, c.homePath)
	if err != nil {
		return err
	}
	c.lastPoint = ""
	c.logger.Printf("%s crawler started", c.fullName)
	return nil
}

/* 执行一次题库爬取
 * limit: 一次最多爬取题目数
 */
func (c *SYZOJ) Update(limit int) (public.FileList, error) {
	c.logger.Printf("Updating %s", c.fullName)
	c.fileList = make(map[string][]byte)
	problemPage, err := public.GetDocument(nil, c.homeUrl+"/problems")
	if err != nil {
		return nil, err
	}
	list := problemPage.Find(".ui.pagination.menu")
	maxPage := 0
	if list.Size() > 0 {
		for i := range list.Eq(0).Nodes {
			//tt:=list.Eq(i).Text()
			for _, j := range strings.Split(list.Eq(i).Text(), "\n") {
				t, err := strconv.Atoi(strings.Trim(j, " "))
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
		return nil, fmt.Errorf("maxPage error: %d", maxPage)
	}
	newPList := make([]public.ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		problemListPage, err := public.GetDocument(nil, fmt.Sprintf("%s/problems?page=%d", c.homeUrl, i))
		if err != nil {
			return nil, err
		}
		list := problemListPage.Find(`[style="vertical-align: middle; "]`)
		for j := range list.Nodes {
			p := public.ProblemListItem{Title: strings.Replace(list.Eq(j).Text(), "\n", "", -1)}
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
	c.lastPoint = public.DownloadProblems(newPList, c.oldPList, limit, c.lastPoint, c.getProblem)
	err = public.WriteFiles(newPList, c.fileList, c.homePath)
	if err != nil {
		return nil, err
	}
	c.oldPList = make(map[string]bool)
	for _, i := range newPList {
		c.oldPList[i.Pid] = true
	}
	return c.fileList, nil
}

func (c *SYZOJ) Stop() {
	c.logger.Println(c.fullName + " crawler stopped")
}

func (c *SYZOJ) getProblem(i *public.ProblemListItem) error {
	c.logger.Println("start getting problem ", i.Pid)
	i.Data = nil
	res, err := public.SafeGet(nil, fmt.Sprintf("%s/problem/%s/export", c.homeUrl, i.Pid))
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()
	data := &syzojExportProblem{}
	err = json.Unmarshal(b, data)
	if err != nil {
		return err
	}
	if !data.Success {
		return err
	}
	i.Data = &public.Problem{}
	i.Data.DescriptionType = "markdown"
	i.Data.Title = i.Title
	i.Data.Time = data.Obj.TimeLimit
	i.Data.Memory = data.Obj.MemoryLimit
	i.Data.Url = c.homeUrl + "/problem/" + i.Pid
	switch data.Obj.Type {
	case "traditional":
		i.Data.Judge = "传统"
	case "submit-answer":
		i.Data.Judge = "提交答案"
	case "interaction":
		i.Data.Judge = "交互"
	}
	for _, k := range data.Obj.Tags {
		if k == "Special Judge" {
			i.Data.Judge += " Special Judge"
			break
		}
	}
	i.Data.Description = fmt.Sprintf(
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
	t, err := public.DownloadImage(nil, i.Data.Description, c.homePath+i.Pid+"/img/", c.fileList, c.homeUrl+"/problem/"+i.Pid+"/", c.homeUrl)
	if err == nil {
		i.Data.Description = t
	}
	return nil
}
