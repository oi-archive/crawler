package main

import (
	"context"
	. "crawler/plugin/public"
	"crawler/rpc"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"strconv"
)

var client rpc.APIClient

var debugMode bool

var oldPList map[string]string

func Start(info *rpc.Info) error {
	oldPList = make(map[string]string)
	err := InitPList(oldPList, info, client)
	if err != nil {
		return err
	}
	log.Println(info.Name + " crawler started")
	return nil
}

type ProblemListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Result []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Tags      string `json:"tags"`
			IsVisible bool   `json:"isVisible"`
			Source    string `json:"source"`
		} `json:"result"`
		Count int `json:"count"`
	} `json:"data"`
}
type ProblemResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ID                            string `json:"id"`
		Title                         string `json:"title"`
		Body                          string `json:"body"`
		Tags                          string `json:"tags"`
		IsVisible                     bool   `json:"isVisible"`
		Source                        string `json:"source"`
		TimeLimitationPerCaseInMs     int    `json:"timeLimitationPerCaseInMs"`
		MemoryLimitationPerCaseInByte int    `json:"memoryLimitationPerCaseInByte"`
	} `json:"data"`
}

type SampleResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	} `json:"data"`
}

var fileList map[string][]byte

func Update(info *rpc.Info, src string) (FileList, error) {
	log.Println("Updating " + info.Name)
	limit := 100
	if debugMode {
		limit = 5
	}
	fileList = make(map[string][]byte)
	b, err := Download(nil, "http://api.oj.joyoi.cn/api/problem/all?tag=&title=&page=1")
	check(err)
	plRes := &ProblemListResponse{}
	err = json.Unmarshal(b, plRes)
	check(err)
	if plRes.Code != 200 {
		log.Panicf("Download ProblemList Error: code = %d, Msg = %s", plRes.Code, plRes.Msg)
	}
	maxPage := plRes.Data.Count
	if maxPage <= 0 || maxPage > 1000 {
		log.Panicln("maxPage error: ", maxPage)
	}
	if debugMode {
		maxPage = 2
	}
	newPList := make([]ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		b, err = Download(nil, "http://api.oj.joyoi.cn/api/problem/all?tag=&title=&page="+strconv.Itoa(i))
		check(err)
		plRes = &ProblemListResponse{}
		err = json.Unmarshal(b, plRes)
		check(err)
		for _, j := range plRes.Data.Result {
			if j.Source == src && j.IsVisible {
				newPList = append(newPList, ProblemListItem{Pid: j.ID, Title: j.Title})
			}
		}
	}
	uList := ChooseUpdateProblem(newPList, oldPList, limit)
	for k := range newPList {
		i := &newPList[k]
		if _, ok := uList[i.Pid]; !ok {
			continue
		}
		if debugMode {
			log.Println("start getting problem ", i.Pid)
		}
		b, err = Download(nil, "http://api.oj.joyoi.cn/api/problem/"+i.Pid)
		if err != nil {
			log.Println("error when parsing problem ", i.Pid, " :", err)
			continue
		}
		res := &ProblemResponse{}
		err = json.Unmarshal(b, res)
		if err != nil {
			log.Println("error when parsing problem ", i.Pid, " :", err)
			continue
		}
		if res.Code != 200 {
			if debugMode {
				log.Printf("Download Problem %s Error: code = %d, Msg = %s", i.Pid, res.Code, res.Msg)
			}
			continue
		}
		if res.Data.Source != src || !res.Data.IsVisible {
			continue
		}
		i.Data = &Problem{}
		i.Data.Time = res.Data.TimeLimitationPerCaseInMs
		i.Data.Memory = res.Data.MemoryLimitationPerCaseInByte / 1024 / 1024
		i.Data.Title = i.Title
		i.Data.Url = "http://www.joyoi.cn/problem/" + i.Pid
		if src == "Local" {
			i.Data.DescriptionType = "markdown"
			i.Data.Description = res.Data.Body
			if len(i.Data.Description) > 0 && i.Data.Description[0] != '#' {
				i.Data.Description = "# \n" + i.Data.Description
			}
			b, err = Download(nil, "http://api.oj.joyoi.cn/api/problem/"+i.Pid+"/testcase/all?type=Sample&showContent=true&contestId=")
			if err == nil {
				spRes := &SampleResponse{}
				err = json.Unmarshal(b, spRes)
				if err == nil && spRes.Code == 200 && len(spRes.Data) > 0 {
					sample := `# 样例数据
<style>
        table,table tr th, table tr td { border:1px solid #0094ff; }
        table { width: 200px; min-height: 25px; line-height: 25px; text-align: center; border-collapse: collapse;}   
    </style>
<table>
	<tr>
		<td>输入样例</td>
		<td>输出样例</td>
	</tr>
`
					for _, j := range spRes.Data {
						sample += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", j.Input, j.Output)
					}
					sample += "</table>\n"
					i.Data.Description += sample
				}
			}
		} else {
			i.Data.DescriptionType = "html_final"
			i.Data.Description = res.Data.Body
		}
		d2, err := DownloadImage(nil, i.Data.Description, info.Id+"/"+i.Pid+"/img/", fileList, "http://www.joyoi.cn/problem/"+i.Pid+"/", "http://www.joyoi.cn")
		if err == nil {
			i.Data.Description = d2
		}
	}
	err = WriteFiles(newPList, fileList, info.Id+"/")
	if err != nil {
		return nil, err
	}
	return fileList, nil
}

func runUpdate(info *rpc.Info, src string) {
	file, err := Update(info, src)
	if err != nil {
		log.Println("Update Error")
		return
	}
	r, err := client.Update(context.Background(), &rpc.UpdateRequest{Info: info, File: file})
	if err != nil {
		log.Printf("Submit update failed: %v", err)
		return
	}
	if !r.Ok {
		log.Println("Submit update failed")
		return
	}
	log.Println("Submit update successfully")
}
func runOJ(info *rpc.Info, src string) {
	err := Start(info)
	if err != nil {
		log.Panicln(err)
	}
	r, err := client.Register(context.Background(), &rpc.RegisterRequest{Info: info})
	if err != nil {
		log.Fatalf("could not register: %v", err)
	}

	debugMode = r.DebugMode
	runUpdate(info, src)
}
func main() {
	conn, err := grpc.Dial("127.0.0.1:27381", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client = rpc.NewAPIClient(conn)
	runOJ(&rpc.Info{Id: "joyoi", Name: "JoyOI"}, "Local")
	runOJ(&rpc.Info{Id: "codevs", Name: "CodeVS"}, "CodeVS")
}

func check(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
