/*
 爬虫通用工具包
*/
package public

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// SafeGet 是 http.Get 的简单封装，会在产生错误时重试 2 次，若重试全部失败，则返回最后一次的错误
func SafeGet(url string) (res *http.Response, err error) {
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

// Download 用于下载一个 url 中的内容
func Download(url string) ([]byte, error) {
	res, err := SafeGet(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// 返回输入 url 的 goquery.Document
func GetDocument(url string) (*goquery.Document, error) {
	res, err := SafeGet(url)
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

// 返回输入 url 的文件扩展名（若获取失败则返回空字符串）
func getFileExtension(url string) string {
	a := strings.Split(url, ".")
	if len(a) <= 1 {
		return ""
	}
	b := strings.Split(a[len(a)-1], "?")
	return b[0]
}

var urlRule = regexp.MustCompile(`(https?|ftp|file)://[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`)

// 判断是否为一个合法的完整 url
func IsUrl(url string) bool {
	return urlRule.MatchString(url)
}

// 解析文档中的图片，下载后保存至 fileList 中。
// text: 待解析的文档; prefix: 文件系统路径前缀;
// fileList: 文件表; url: 文档链接，用于相对路径的处理，若不需要则置空
// 返回替换图片链接后的文档
func DownloadImage(text string, prefix string, fileList map[string][]byte, url string) (string, error) {
	rule := regexp.MustCompile(`!\[.*?]\(.+?\)`)
	r2 := regexp.MustCompile(`\(.+?\)`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		match := r2.FindString(x)
		match = match[1 : len(match)-1]
		file, err := Download(match)
		if err != nil {
			match = url + match
			file, err = Download(match)
			if err != nil {
				log.Printf("download image %s error", match)
				return x
			}
		}
		b64 := base64.URLEncoding.EncodeToString([]byte(match))
		path := prefix + b64 + "." + getFileExtension(match)
		fileList[path] = file
		return r2.ReplaceAllString(x, "(source/"+path+")")
	})
	rule = regexp.MustCompile(`<img[^>]+src\s*=\s*['"]([^'"]+)['"][^>]*>`)
	r2 = regexp.MustCompile(`['"][^'"]+['"]`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		//log.Println(x)
		match := r2.FindString(x)
		match = match[1 : len(match)-1]
		file, err := Download(match)
		if err != nil {
			match = url + match
			file, err = Download(match)
			if err != nil {
				log.Printf("download image %s error", match)
				return x
			}
		}
		b64 := base64.URLEncoding.EncodeToString([]byte(match))
		path := prefix + b64 + "." + getFileExtension(match)
		fileList[path] = file
		return r2.ReplaceAllString(x, `"source/`+path+`"`)
	})
	return text, nil
}
