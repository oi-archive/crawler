package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/oi-archive/crawler/plugin/public"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
)

const homePath = "bzoj/"

type config struct {
	Username string
	Password string
}

var cfg config

func login(c *http.Client) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	c.Jar = jar
	b, err := public.PostAndRead(c, "https://lydsy.com/JudgeOnline/login.php", url.Values{"user_id": {cfg.Username}, "password": {cfg.Password}})
	if err != nil {
		return err
	}
	if strings.Contains(string(b), "alert") {
		return fmt.Errorf("Login error:%s", string(b))
	}
	return nil
}

type addUATransport struct {
	T http.RoundTripper
}

func (adt *addUATransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", `User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/76.0.3809.87 Chrome/76.0.3809.87 Safari/537.36`)
	return adt.T.RoundTrip(req)
}

func newAddUATransport(T http.RoundTripper) *addUATransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &addUATransport{T}
}

var oldPList map[string]bool
var lastPoint string

func Start() error {
	oldPList = make(map[string]bool)
	err := public.InitPList(oldPList, homePath)
	if err != nil {
		return err
	}
	lastPoint = ""
	b, err := ioutil.ReadFile("./config/bzoj")
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return err
	}
	c := &http.Client{Transport: newAddUATransport(nil)}
	err = login(c)
	if err != nil {
		return err
	}
	log.Println("BZOJ crawler started")
	return nil
}

var fileList map[string][]byte

func Update(limit int) (map[string][]byte, error) {
	fileList = make(map[string][]byte)
	c := &http.Client{Transport: newAddUATransport(nil)}
	err := login(c)
	if err != nil {
		return nil, err
	}
	problemPage, err := public.GetDocument(c, "https://lydsy.com/JudgeOnline/problemset.php")
	if err != nil {
		return nil, err
	}
	list := problemPage.Find("h3")
	maxPage := 0
	if list.Size() > 0 {
		for i := range list.Eq(0).Nodes {
			tt := list.Eq(i).Text()
			for _, j := range strings.Split(tt, string(rune(160))) {
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
		return nil, fmt.Errorf("maxPage error: %d", maxPage)
	}
	newPList := make([]public.ProblemListItem, 0)
	for i := 1; i <= maxPage; i++ {
		problemListPage, err := public.GetDocument(c, fmt.Sprintf("https://lydsy.com/JudgeOnline/problemset.php?page=%d", i))
		if err != nil {
			return nil, err
		}
		list := problemListPage.Find(`.evenrow,.oddrow`)
		list.Each(func(_ int, s *goquery.Selection) {
			t := s.Nodes[0]
			// TODO: 各类错误处理
			if t.FirstChild.NextSibling.NextSibling == nil {
				return
			}
			p := public.ProblemListItem{}
			j := t.FirstChild.NextSibling
			p.Pid = j.FirstChild.Data
			j = j.NextSibling.FirstChild.FirstChild
			p.Title = j.Data
			newPList = append(newPList, p)
		})
	}
	lastPoint = public.DownloadProblems(newPList, oldPList, limit, lastPoint, func(i *public.ProblemListItem) error {
		log.Println("start getting problem ", i.Pid)
		i.Data = nil
		page, err := public.GetDocument(c, `https://lydsy.com/JudgeOnline/problem.php?id=`+i.Pid)
		if err != nil {
			log.Printf("解析题目%s时产生错误：下载题面失败", i.Pid)
			return err
		}
		t := page.Find(".content").Nodes
		if len(t) < 7 {
			log.Printf("解析题目%s时产生错误：无法获取conetnt对象", i.Pid)
			return err
		}
		i.Data = &public.Problem{}
		i.Data.DescriptionType = "html"
		i.Data.Time, err = strconv.Atoi(strings.Split(page.Find(`body > center:nth-child(3) > span:nth-child(2)`).Nodes[0].NextSibling.Data, " ")[0])
		i.Data.Time *= 1000
		if err != nil {
			i.Data.Time = 0
		}
		i.Data.Memory, err = strconv.Atoi(strings.Split(page.Find(`body > center:nth-child(3) > span:nth-child(3)`).Nodes[0].NextSibling.Data, " ")[0])
		if err != nil {
			i.Data.Memory = 0
		}
		if len(page.Find(`body > center:nth-child(3) > span.red`).Nodes) == 0 {
			i.Data.Judge = "传统"
		} else {
			i.Data.Judge = "传统 Special Judge"
		}

		i.Data.Url = `https://lydsy.com/JudgeOnline/problem.php?id=` + i.Pid
		i.Data.Title = i.Title
		i.Data.Description = fmt.Sprintf(`
# Description

%s

# Input

%s

# Output

%s

# Sample Input

%s

# Sample Output

%s

# Hint

%s

# Source

%s

`, node2html(t[0]), node2html(t[1]), node2html(t[2]), node2html(t[3]), node2html(t[4]), node2html(t[5]), node2html(t[6]))
		d2, err := public.DownloadImage(c, i.Data.Description, homePath+i.Pid+"/img/", fileList, "https://lydsy.com/JudgeOnline/", "https://lydsy.com")
		if err == nil {
			i.Data.Description = d2
		}
		return nil
	})
	err = public.WriteFiles(newPList, fileList, homePath)
	if err != nil {
		return nil, err
	}
	return fileList, nil
}

func Stop() {

}

func Name() string {
	return "BZOJ"
}

func node2html(x *html.Node) string {
	var b bytes.Buffer
	_ = html.Render(&b, x)
	return b.String()
}
