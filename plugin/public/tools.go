/*
 爬虫通用工具包
*/
package public

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
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

// SafeGet 是 http.Get 的简单封装，会在产生错误时重试 2 次，若重试全部失败，则返回最后一次的错误
func SafeGet(c *http.Client, url string) (res *http.Response, err error) {
	time.Sleep(50 * time.Millisecond)
	for i := 1; i <= 3; i++ {
		if c == nil {
			res, err = http.Get(url)
		} else {
			res, err = c.Get(url)
		}
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

// SafePost 是 http.Post 的简单封装，会在产生错误时重试 2 次，若重试全部失败，则返回最后一次的错误
func SafePost(c *http.Client, url string, form url.Values) (res *http.Response, err error) {
	time.Sleep(50 * time.Millisecond)
	for i := 1; i <= 3; i++ {
		res, err = c.PostForm(url, form)
		if err != nil {
			time.Sleep(time.Millisecond * 50)
			continue
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf("post %s error,status code = %d", url, res.StatusCode)
			time.Sleep(time.Millisecond * 50)
			continue
		}
		return res, nil
	}
	return nil, err
}

// Download 用于下载一个 url 中的内容
func Download(c *http.Client, url string) ([]byte, error) {
	res, err := SafeGet(c, url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func PostAndRead(c *http.Client, url string, form url.Values) ([]byte, error) {
	res, err := SafePost(c, url, form)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// 返回输入 url 的 goquery.Document
func GetDocument(c *http.Client, url string) (*goquery.Document, error) {
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
func DownloadImage(c *http.Client, text string, prefix string, fileList map[string][]byte, url1 string, url2 string) (string, error) {
	rule := regexp.MustCompile(`!\[.*?]\((.+?)\)`)
	r2 := regexp.MustCompile(`\(.+?\)`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		match := r2.FindString(x)
		match = match[1 : len(match)-1]
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
				log.Printf("download image %s error", match)
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
		return r2.ReplaceAllString(x, "(source/"+path+")")
	})
	rule = regexp.MustCompile(`<img[^>]+src\s*=\s*['"]([^'"]+)['"][^>]*>`)
	r2 = regexp.MustCompile(`src\s*=\s*['"]([^'"]+)['"]`)
	r3 := regexp.MustCompile(`['"]([^'"]+)['"]`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		//log.Println(x)
		match2 := r2.FindString(x)
		match := r3.FindString(match2)
		match = match[1 : len(match)-1]
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
				log.Printf("download image %s error", match)
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
		return r2.ReplaceAllString(x, r3.ReplaceAllString(match2, `"source/`+path+`"`))
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

func DownloadProblems(newPList ProblemList, oldPList map[string]bool, limit int, lastPoint string, getProblem func(*ProblemListItem) error) string {
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
	cnt := 0
	for k := range newPList {
		i := &newPList[k]
		if _, ok := oldPList[i.Pid]; !ok {
			cnt++
			err := f(i)
			if err != nil {
				log.Printf("爬取题目%s时出现错误:%v", i.Pid, err)
			}
		}
	}
	start := -1
	for k := range newPList {
		if newPList[k].Pid == lastPoint {
			start = k
			break
		}
	}
	for k := start + 1; cnt < limit; k++ {
		if k >= len(newPList) {
			k = 0
		}
		i := &newPList[k]
		lastPoint = i.Pid
		cnt++
		err := f(i)
		if err != nil {
			log.Printf("爬取题目%s时出现错误:%v", i.Pid, err)
		}
	}
	return lastPoint
}

func InitPList(oldPList map[string]bool, homePath string) error {
	b, err := ioutil.ReadFile("../source/" + homePath + "problemlist.json")
	if err != nil {
		return err
	}
	x := ProblemList{}
	err = json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	for _, i := range x {
		oldPList[i.Pid] = true
	}
	return nil
}

func Node2html(x *html.Node) string {
	var b bytes.Buffer
	_ = html.Render(&b, x)
	return b.String()
}
