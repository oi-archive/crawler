package main

import (
	"context"
	. "crawler/plugin/public"
	"crawler/rpc"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"time"
)

var client rpc.APIClient

const PID = "lutece"
const NAME = "Lutece"
const homePath = PID + "/"

var info *rpc.Info
var debugMode bool

var oldPList map[string]string

func Start() error {
	oldPList = make(map[string]string)
	err := InitPList(oldPList, info, client)
	if err != nil {
		return err
	}
	log.Println(NAME + " crawler started")
	return nil
}

type Request struct {
	OperationName string `json:"operationName"`
	Query         string `json:"query"`
	Variables     struct {
		Filter string `json:"filter"`
		Page   int    `json:"page"`
		Slug   string `json:"slug"`
	} `json:"variables"`
}
type ProblemListResponse struct {
	Data struct {
		ProblemList struct {
			MaxPage     int
			ProblemList []struct {
				Title string
				Slug  string
			}
		}
	}
}

type ProblemResponse struct {
	Data struct {
		Problem struct {
			Title          string
			Content        string
			StandardInput  string
			StandardOutput string
			Constraints    string
			Note           string
			Limitation     struct {
				TimeLimit   int
				MemoryLimit int
			}
			Samples struct {
				SampleList []struct {
					InputContent  string
					OutputContent string
				}
			}
			Source string
		}
	}
}

var fileList map[string][]byte

func Update() (FileList, error) {
	log.Println("Updating " + NAME)
	limit := 50
	if debugMode {
		limit = 5
	}
	c := &HttpConfig{Client: nil, SleepTime: 500 * time.Millisecond}
	fileList = make(map[string][]byte)
	plReq := Request{OperationName: "ProblemListGQL", Query: `query ProblemListGQL($page: Int!, $filter: String) {
  problemList(page: $page, filter: $filter) {
    maxPage
  }
}
`}
	plReq.Variables.Page = 1
	b, err := json.Marshal(plReq)
	check(err)
	b, err = PostAndRead(c, "https://acm.uestc.edu.cn/graphql", "application/json", b)
	check(err)
	plRes := &ProblemListResponse{}
	err = json.Unmarshal(b, plRes)
	check(err)
	maxPage := plRes.Data.ProblemList.MaxPage
	if maxPage <= 0 || maxPage > 1000 {
		log.Panicln("maxPage error: ", maxPage)
	}
	if debugMode {
		maxPage = 2
	}
	newPList := make([]ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		plReq = Request{OperationName: "ProblemListGQL", Query: `query ProblemListGQL($page: Int!, $filter: String) {
  problemList(page: $page, filter: $filter) {
    problemList {
      title
      slug
    }
  }
}
`}
		plReq.Variables.Page = i
		b, err = json.Marshal(plReq)
		check(err)
		b, err = PostAndRead(c, "https://acm.uestc.edu.cn/graphql", "application/json", b)
		check(err)
		plRes = &ProblemListResponse{}
		err = json.Unmarshal(b, plRes)
		check(err)
		for _, j := range plRes.Data.ProblemList.ProblemList {
			newPList = append(newPList, ProblemListItem{Pid: j.Slug, Title: j.Title})
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
		req := &Request{OperationName: "ProblemDetailGQL", Query: `query ProblemDetailGQL($slug: String!) {
  problem(slug: $slug) {
    title
    content
    standardInput
    standardOutput
    constraints
    resources
    note
    limitation {
      timeLimit
      memoryLimit
    }
    samples {
      sampleList {
        inputContent
        outputContent
      }
    }
  }
}`}
		req.Variables.Slug = i.Pid
		b, err := json.Marshal(req)
		if err != nil {
			log.Println("error when parsing problem ", i.Pid, " :", err)
			continue
		}
		b, err = PostAndRead(c, "https://acm.uestc.edu.cn/graphql", "application/json", b)
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
		i.Data = &Problem{}
		i.Data.Time = res.Data.Problem.Limitation.TimeLimit
		i.Data.Memory = res.Data.Problem.Limitation.MemoryLimit
		i.Data.Title = i.Title
		i.Data.Url = fmt.Sprintf("https://acm.uestc.edu.cn/problem/%s/description", i.Pid)
		i.Data.Judge = "传统"
		sample := `<style>
        table,table tr th, table tr td { border:1px solid #0094ff; }
        table { width: 200px; min-height: 25px; line-height: 25px; text-align: center; border-collapse: collapse;}   
    </style>
<table>
	<tr>
		<td>Input</td>
		<td>Output</td>
	</tr>
`
		for _, j := range res.Data.Problem.Samples.SampleList {
			sample += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", j.InputContent, j.OutputContent)
		}
		sample += "</table>\n"
		i.Data.DescriptionType = "html"
		i.Data.Description = fmt.Sprintf(`
# Content

%s

# Standard Input

%s

# Standard Output

%s

# Samples

%s

# Constraints

%s

# Note

%s

# Source

%s
`, res.Data.Problem.Content, res.Data.Problem.StandardInput, res.Data.Problem.StandardOutput, sample, res.Data.Problem.Constraints, res.Data.Problem.Note, res.Data.Problem.Source)
		d2, err := DownloadImage(c, i.Data.Description, homePath+i.Pid+"/img/", fileList, "https://acm.uestc.edu.cn/problem/"+i.Pid+"/description", "https://acm.uestc.edu.cn")
		if err == nil {
			i.Data.Description = d2
		}
	}
	err = WriteFiles(newPList, fileList, homePath)
	if err != nil {
		return nil, err
	}
	return fileList, nil
}

func runUpdate() {
	file, err := Update()
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
func main() {
	conn, err := grpc.Dial("127.0.0.1:27381", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	info = &rpc.Info{Id: PID, Name: NAME}
	client = rpc.NewAPIClient(conn)
	err = Start()
	if err != nil {
		log.Panicln(err)
	}
	r, err := client.Register(context.Background(), &rpc.RegisterRequest{Info: info})
	if err != nil {
		log.Fatalf("could not register: %v", err)
	}

	debugMode = r.DebugMode
	runUpdate()
}

func check(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
