package syzoj

import (
	"context"
	. "crawler/plugin/public"
	"crawler/rpc"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
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
	info      *rpc.Info
	client    rpc.APIClient
	homeUrl   string
	homePath  string
	fileList  FileList
	oldPList  map[string]string
	debugMode bool
	conn      *grpc.ClientConn
}

func (c *SYZOJ) Start(info *rpc.Info, hu string) error {
	c.info = info
	c.homeUrl = hu
	c.homePath = c.info.Id + "/"
	var err error
	c.conn, err = grpc.Dial("127.0.0.1:27381", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c.client = rpc.NewAPIClient(c.conn)
	c.oldPList = make(map[string]string)
	err = InitPList(c.oldPList, c.info, c.client)
	if err != nil {
		return err
	}
	log.Printf("%s crawler started", c.info.Name)
	r, err := c.client.Register(context.Background(), &rpc.RegisterRequest{Info: info})
	if err != nil {
		log.Fatalf("could not register: %v", err)
	}
	log.Println("Is debug mode: ", r.DebugMode)
	c.debugMode = r.DebugMode
	return nil
}

/* 执行一次题库爬取
 * limit: 一次最多爬取题目数
 */
func (c *SYZOJ) Update(limit int) error {
	if c.debugMode {
		limit = 5
	}
	log.Printf("Updating %s", c.info.Name)
	c.fileList = make(map[string][]byte)
	problemPage, err := GetDocument(nil, c.homeUrl+"/problems")
	if err != nil {
		return err
	}
	list := problemPage.Find(".ui.pagination.menu")
	maxPage := 0
	if list.Size() > 0 {
		for i := list.Nodes[0].FirstChild; i != nil; i = i.NextSibling {
			if i.FirstChild != nil {
				t, err := strconv.Atoi(strings.Trim(i.FirstChild.Data, "\n\r\t "))
				if err == nil && t > maxPage {
					maxPage = t
				}
			}
		}
	}
	if maxPage <= 0 || maxPage >= 500 {
		return fmt.Errorf("maxPage error: %d", maxPage)
	}
	if c.debugMode {
		maxPage = 2
	}
	newPList := make([]ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		problemListPage, err := GetDocument(nil, fmt.Sprintf("%s/problems?page=%d", c.homeUrl, i))
		if err != nil {
			return err
		}
		list := problemListPage.Find(`[style^=vertical-align]`)
		for j := range list.Nodes {
			p := ProblemListItem{Title: strings.Replace(list.Eq(j).Text(), "\n", "", -1)}
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
	log.Println(len(newPList))
	DownloadProblems(newPList, c.oldPList, limit, c.getProblem)
	err = WriteFiles(newPList, c.fileList, c.homePath)
	if err != nil {
		return err
	}
	c.oldPList = make(map[string]string)
	for _, i := range newPList {
		c.oldPList[i.Pid] = i.Title
	}
	r, err := c.client.Update(context.Background(), &rpc.UpdateRequest{Info: c.info, File: c.fileList})
	if err != nil {
		log.Printf("Submit update failed: %v", err)
		return err
	}
	if !r.Ok {
		log.Println("Submit update failed")
		return err
	}
	log.Println("Submit update successfully")
	return nil
}

func (c *SYZOJ) Stop() {
	c.conn.Close()
	log.Println(c.info.Name + " crawler stopped")
}

func (c *SYZOJ) getProblem(i *ProblemListItem) error {
	if c.debugMode {
		log.Println("start getting problem ", i.Pid)
	}
	i.Data = nil
	res, err := SafeGet(nil, fmt.Sprintf("%s/problem/%s/export", c.homeUrl, i.Pid))
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
	i.Data = &Problem{}
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
	t, err := DownloadImage(nil, i.Data.Description, c.homePath+i.Pid+"/img/", c.fileList, c.homeUrl+"/problem/"+i.Pid+"/", c.homeUrl)
	if err == nil {
		i.Data.Description = t
	}
	return nil
}
