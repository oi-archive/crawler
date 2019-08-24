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

func Download(url string) ([]byte, error) {
	res, err := SafeGet(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

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

func getFileExtension(url string) string {
	a := strings.Split(url, ".")
	if len(a) <= 1 {
		return ""
	}
	b := strings.Split(a[len(a)-1], "?")
	return b[0]
}

func DownloadImage(text string, prefix string, fileList map[string][]byte) (string, error) {
	rule := regexp.MustCompile(`!\[.*?]\(.+?\)`)
	r2 := regexp.MustCompile(`\(.+?\)`)
	text = rule.ReplaceAllStringFunc(text, func(x string) string {
		match := r2.FindString(x)
		match = match[1 : len(match)-1]
		file, err := Download(match)
		if err != nil {
			log.Printf("download image %s error", match)
			return x
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
			log.Printf("download image %s error", match)
			return x
		}
		b64 := base64.URLEncoding.EncodeToString([]byte(match))
		path := prefix + b64 + "." + getFileExtension(match)
		fileList[path] = file
		return r2.ReplaceAllString(x, `"source/`+path+`"`)
	})
	return text, nil
}
