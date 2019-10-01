package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/oi-archive/crawler/plugin/public"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
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

var logger *log.Logger

var oldPList map[string]bool
var lastPoint string

func Start(logg *log.Logger) error {
	logger = logg
	oldPList = make(map[string]bool)
	err := public.InitPList(oldPList, homePath)
	if err != nil {
		return err
	}
	lastPoint = ""
	b, err := ioutil.ReadFile("./config/bzoj.json")
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
	logger.Println("BZOJ crawler started")
	return nil
}

var fileList map[string][]byte

func Update(limit int) (public.FileList, error) {
	logger.Println("Updating BZOJ")
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
			if len(s.Nodes) == 0 {
				return
			}
			t := s.Nodes[0]
			if t == nil || t.FirstChild == nil || t.FirstChild.NextSibling == nil || t.FirstChild.NextSibling.NextSibling == nil {
				return
			}
			p := public.ProblemListItem{}
			j := t.FirstChild.NextSibling
			p.Pid = j.FirstChild.Data
			if j.NextSibling == nil || j.NextSibling.FirstChild == nil || j.NextSibling.FirstChild.FirstChild == nil {
				return
			}
			j = j.NextSibling.FirstChild.FirstChild
			p.Title = j.Data
			newPList = append(newPList, p)
		})
	}
	lastPoint = public.DownloadProblems(newPList, oldPList, limit, lastPoint, func(i *public.ProblemListItem) (err error) {
		logger.Println("start getting problem ", i.Pid)
		i.Data = nil
		page, err := public.GetDocument(c, `https://lydsy.com/JudgeOnline/problem.php?id=`+i.Pid)
		if err != nil {
			logger.Printf("解析题目%s时产生错误：下载题面失败", i.Pid)
			return err
		}
		t := page.Find(".content").Nodes
		if len(t) < 7 {
			logger.Printf("解析题目%s时产生错误：无法获取conetnt对象", i.Pid)
			return err
		}
		pos := "3"
		if strings.Contains(public.Node2html(page.Nodes[0]), `class="notice"`) {
			pos = "4"
		}
		i.Data = &public.Problem{}
		i.Data.DescriptionType = "html"
		i.Data.Time, err = strconv.Atoi(strings.Split(page.Find(fmt.Sprintf(`body > center:nth-child(%s) > span:nth-child(2)`, pos)).Nodes[0].NextSibling.Data, " ")[0])
		i.Data.Time *= 1000
		if err != nil {
			i.Data.Time = 0
		}
		i.Data.Memory, err = strconv.Atoi(strings.Split(page.Find(fmt.Sprintf(`body > center:nth-child(%s) > span:nth-child(3)`, pos)).Nodes[0].NextSibling.Data, " ")[0])
		if err != nil {
			i.Data.Memory = 0
		}
		if len(page.Find(fmt.Sprintf(`body > center:nth-child(%s) > span.red`, pos)).Nodes) == 0 {
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

`, public.Node2html(t[0]), public.Node2html(t[1]), public.Node2html(t[2]), public.Node2html(t[3]), public.Node2html(t[4]), public.Node2html(t[5]), public.Node2html(t[6]))
		r := regexp.MustCompile(`<p>[\s]*`)
		i.Data.Description = r.ReplaceAllString(i.Data.Description, `<p>`)
		i.Data.Description = strings.ReplaceAll(i.Data.Description, "<br>\n", "<br>")
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
