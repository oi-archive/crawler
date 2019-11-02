/*
 爬虫通用工具包
*/
package public

import (
	"bytes"
	"context"
	"crawler/rpc"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Problem struct {
	Time            int    `json:"time"`
	Memory          int    `json:"memory"`
	Title           string `json:"title"`
	Judge           string `json:"judge"`
	Url             string `json:"url"`
	Description     string `json:"-"`
	DescriptionType string `json:"description_type"`
}

type ProblemListItem struct {
	Title string   `json:"title"`
	Pid   string   `json:"pid"`
	Data  *Problem `json:"-"`
}

type ProblemList []ProblemListItem

type FileList map[string][]byte

type HttpConfig struct {
	Client    *http.Client
	SleepTime time.Duration
}

var DefaultHttpConfig = &HttpConfig{Client: nil, SleepTime: 200 * time.Millisecond}

// SafeGet 是 http.Get 的简单封装，会在产生错误时重试 2 次，若重试全部失败，则返回最后一次的错误
func SafeGet(c *HttpConfig, url string) (res *http.Response, err error) {
	if c == nil {
		c = DefaultHttpConfig
	}
	time.Sleep(c.SleepTime)
	for i := 1; i <= 3; i++ {
		if c.Client == nil {
			res, err = http.Get(url)
		} else {
			res, err = c.Client.Get(url)
		}
		if err != nil {
			time.Sleep(c.SleepTime)
			continue
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf("get %s error,status code = %d", url, res.StatusCode)
			time.Sleep(c.SleepTime)
			continue
		}
		return res, nil
	}
	return nil, err
}

// SafePost 是 http.PostForm 的简单封装，会在产生错误时重试 2 次，若重试全部失败，则返回最后一次的错误
func SafePost(c *HttpConfig, url string, contentType string, data []byte) (res *http.Response, err error) {
	if c == nil {
		c = DefaultHttpConfig
	}
	time.Sleep(c.SleepTime)
	for i := 1; i <= 3; i++ {
		if c.Client == nil {
			res, err = http.Post(url, contentType, bytes.NewReader(data))
		} else {
			res, err = c.Client.Post(url, contentType, bytes.NewReader(data))
		}
		if err != nil {
			time.Sleep(c.SleepTime)
			continue
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf("post %s error,status code = %d", url, res.StatusCode)
			time.Sleep(c.SleepTime)
			continue
		}
		return res, nil
	}
	return nil, err
}

// SafePostForm 是 http.PostForm 的简单封装，会在产生错误时重试 2 次，若重试全部失败，则返回最后一次的错误
func SafePostForm(c *HttpConfig, url string, form url.Values) (res *http.Response, err error) {
	if c == nil {
		c = DefaultHttpConfig
	}
	time.Sleep(c.SleepTime)
	for i := 1; i <= 3; i++ {
		if c.Client == nil {
			res, err = http.PostForm(url, form)
		} else {
			res, err = c.Client.PostForm(url, form)
		}
		if err != nil {
			time.Sleep(c.SleepTime)
			continue
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf("post %s error,status code = %d", url, res.StatusCode)
			time.Sleep(c.SleepTime)
			continue
		}
		return res, nil
	}
	return nil, err
}

// Download 用于下载一个 url 中的内容
func Download(c *HttpConfig, url string) ([]byte, error) {
	res, err := SafeGet(c, url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// Post 并返回 respose 的 body 的内容
func PostAndRead(c *HttpConfig, url string, contentType string, data []byte) ([]byte, error) {
	res, err := SafePost(c, url, contentType, data)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// PostForm 并返回 respose 的 body 的内容
func PostFormAndRead(c *HttpConfig, url string, form url.Values) ([]byte, error) {
	res, err := SafePostForm(c, url, form)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// 返回输入 url 的 goquery.Document
func GetDocument(c *HttpConfig, url string) (*goquery.Document, error) {
	res, err := SafeGet(c, url)
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

// 返回输入 url 的文件扩展名
func getFileExtension(url string) (string, error) {
	a := strings.Split(url, ".")
	if len(a) <= 1 {
		return "", fmt.Errorf("")
	}
	b := strings.Split(a[len(a)-1], "?")
	if len(b[0]) > 5 {
		return "", fmt.Errorf("")
	}
	return b[0], nil
}

var urlRule = regexp.MustCompile(`(https?|ftp|file)://[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`)

// 判断是否为一个合法的完整 url
func IsUrl(url string) bool {
	return urlRule.MatchString(url)
}

func CalcMD5(x string) string {
	h := md5.Sum([]byte(x))
	return fmt.Sprintf("%x", h)
}

// 解析文档中的图片，下载后保存至 fileList 中。
// c http实例，不需要可置nil; text: 待解析的文档; prefix: 文件系统路径前缀;
// fileList: 文件表; url1,url2: 文档链接和域名链接，用于相对路径的处理，若不需要则置空
// 返回替换图片链接后的文档
func DownloadImage(c *HttpConfig, text string, prefix string, fileList map[string][]byte, url1 string, url2 string) (string, error) {
	rule := regexp.MustCompile(`!\[.*?]\((.+?)\)`)
	r2 := regexp.MustCompile(`\(.+?\)`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		match := r2.FindString(x)
		match = match[1 : len(match)-1]
		if len(match) > 1000 {
			return x
		}
		matchBak := match
		file, err := Download(c, match)
		if err != nil {
			if url1 != "" && match[0] != '/' {
				match = url1 + match
				file, err = Download(c, match)
			} else if url2 != "" && match[0] == '/' {
				match = url2 + match
				file, err = Download(c, match)
			}
			if err != nil {
				log.Printf("Problem %s : download image %s error", url1, matchBak)
				return x
			}
		}
		b64 := base64.URLEncoding.EncodeToString([]byte(match))
		if len(b64) > 200 {
			b64 = CalcMD5(b64)
		}
		path := ""
		ex, err := getFileExtension(match)
		if err == nil {
			path = prefix + b64 + "." + ex
		} else {
			path = prefix + b64
		}
		fileList[path] = file
		return r2.ReplaceAllString(x, "(/source/"+path+")")
	})
	rule = regexp.MustCompile(`<img[^>]+src\s*=\s*['"]([^'"]+)['"][^>]*>`)
	r2 = regexp.MustCompile(`src\s*=\s*['"]([^'"]+)['"]`)
	r3 := regexp.MustCompile(`['"]([^'"]+)['"]`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		//log.Println(x)
		match2 := r2.FindString(x)
		match := r3.FindString(match2)
		match = match[1 : len(match)-1]
		if len(match) > 1000 {
			return x
		}
		matchBak := match
		file, err := Download(c, match)
		if err != nil {
			if url1 != "" && match[0] != '/' {
				match = url1 + match
				file, err = Download(c, match)
			} else if url2 != "" && match[0] == '/' {
				match = url2 + match
				file, err = Download(c, match)
			}
			if err != nil {
				log.Printf("Problem %s : download image %s error", url1, matchBak)
				return x
			}
		}
		b64 := base64.URLEncoding.EncodeToString([]byte(match))
		if len(b64) > 200 {
			b64 = CalcMD5(b64)
		}
		path := ""
		ex, err := getFileExtension(match)
		if err == nil {
			path = prefix + b64 + "." + ex
		} else {
			path = prefix + b64
		}
		fileList[path] = file
		return r2.ReplaceAllString(x, r3.ReplaceAllString(match2, `"/source/`+path+`"`))
	})
	return text, nil
}

// 向文件表写入 problemlist
func WriteProblemList(list ProblemList, fileList FileList, homePath string) error {
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	fileList[homePath+"problemlist.json"] = b
	return nil
}

// 向文件表写入 main.json
func WriteMainJson(path string, p *ProblemListItem, fileList FileList) error {
	b, err := json.Marshal(p.Data)
	if err != nil {
		return err
	}
	fileList[path] = b
	return nil
}

// 向文件表写入文件
func WriteFiles(pList ProblemList, fileList FileList, homePath string) error {
	err := WriteProblemList(pList, fileList, homePath)
	if err != nil {
		return err
	}
	for _, i := range pList {
		if i.Data == nil {
			continue
		}
		nowPath := homePath + i.Pid + "/"
		err = WriteMainJson(nowPath+"main.json", &i, fileList)
		if err != nil {
			log.Println(err)
			continue
		}
		fileList[nowPath+"description.md"] = []byte(i.Data.Description)
	}
	return nil
}

// 选定本次要更新的题目
func ChooseUpdateProblem(newPList ProblemList, oldPList map[string]string, limit int) map[string]bool {
	rand.Seed(time.Now().Unix())
	if limit > len(newPList) {
		limit = len(newPList)
	}
	res := make(map[string]bool)
	if limit == 0 {
		return res
	}
	cnt := 0
	for k := range newPList {
		i := &newPList[k]
		if title, ok := oldPList[i.Pid]; !ok || title != i.Title {
			cnt++
			res[i.Pid] = true
		}
	}
	for cnt < limit {
		i := rand.Intn(len(newPList))
		if _, ok := res[newPList[i].Pid]; ok {
			continue
		}
		cnt++
		res[newPList[i].Pid] = true
	}
	return res
}

func DownloadProblems(newPList ProblemList, oldPList map[string]string, limit int, getProblem func(*ProblemListItem) error) {
	rand.Seed(time.Now().Unix())
	f := func(i *ProblemListItem) (err error) {
		defer func() {
			if perr := recover(); perr != nil {
				log.Println("解析题目时产生异常：", perr)
				i.Data = nil
				err = perr.(error)
			}
		}()
		return getProblem(i)
	}
	if limit > len(newPList) {
		limit = len(newPList)
	}
	if limit == 0 {
		return
	}
	b := make([]bool, len(newPList))
	for i := range b {
		b[i] = true
	}
	cnt := 0
	for k := range newPList {
		i := &newPList[k]
		if title, ok := oldPList[i.Pid]; !ok || title != i.Title {
			b[k] = false
			cnt++
			err := f(i)
			if err != nil {
				i.Data = nil
				log.Printf("爬取题目%s时出现错误:%v", i.Pid, err)
			}
		}
	}
	for cnt < limit {
		i := rand.Intn(len(newPList))
		if !b[i] {
			continue
		}
		b[i] = false
		cnt++
		err := f(&newPList[i])
		if err != nil {
			newPList[i].Data = nil
			log.Printf("爬取题目%s时出现错误:%v", newPList[i].Pid, err)
		}
	}
}

func InitPList(oldPList map[string]string, info *rpc.Info, client rpc.APIClient) error {
	req, err := client.GetProblemlist(context.Background(), info)
	if err != nil {
		return err
	}
	for _, i := range req.Data {
		oldPList[i.Pid] = i.Title
	}
	return nil
}

func Node2html(x *html.Node) string {
	var b bytes.Buffer
	_ = html.Render(&b, x)
	return b.String()
}

func NodeChildren2html(x *html.Node) string {
	s := ""
	for i := x.FirstChild; i != nil; i = i.NextSibling {
		s += Node2html(i)
	}
	return s
}
