// 由于 Tsinsen 将(或已经)于 2019.9,1 关闭，本爬虫为一次性爬虫。

package main

import (
	. "crawler/plugin/public"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var logger *log.Logger

const homePath = "tsinsen/"

var fileList map[string][]byte

func Name() string {
	return "Tsinsen"
}

func Start(logg *log.Logger) error {
	logger = logg
	logger.Println("Tsinsen crawler started")
	return nil
}

func Update(limit int) (FileList, error) {
	logger.Println("Updating Tsinsen")
	fileList = make(FileList)
	pl := make(ProblemList, 0)
	for i := 1000; i <= 1518; i++ {
		p := ProblemListItem{}
		p.Data = &Problem{}
		p.Pid = "A" + strconv.Itoa(i)
		p.Data.Url = "http://www.tsinsen.com/" + p.Pid
		log.Println("start getting problem " + p.Pid)
		page, err := GetDocument(nil, p.Data.Url)
		if err != nil {
			log.Panicln(err)
		}
		p.Title = page.Find(`#ptit`).Nodes[0].FirstChild.Data
		p.Title = p.Title[7:len(p.Title)]
		p.Data.Title = p.Title
		t := page.Find(`#pres > div:nth-child(1) > span:nth-child(1)`).Nodes[0].FirstChild.Data
		t2 := page.Find(`#pres > div:nth-child(1) > span:nth-child(2)`).Nodes[0].FirstChild.Data
		log.Println(p.Title, t, t2)
		p.Data.Time, err = strconv.Atoi(strings.Split(t, ".")[0])
		if err != nil {
			x := 0
			for _, i := range t {
				if i < int32('0') || i > int32('9') {
					break
				}
				x = x*10 + int(i) - int('0')
			}
			p.Data.Time = x
		}
		if !strings.Contains(t, "ms") {
			p.Data.Time *= 1000
		}
		p.Data.Memory, err = strconv.Atoi(strings.Split(t2, ".")[0])
		if err != nil {
			x := 0
			for _, i := range t {
				if i < int32('0') || i > int32('9') {
					break
				}
				x = x*10 + int(i) - int('0')
			}
			p.Data.Memory = x
		}
		if strings.Contains(t, "GB") {
			p.Data.Memory *= 1024
		}
		p.Data.Judge = ""
		p.Data.DescriptionType = "html"
		t3 := page.Find(`#pcont1`).Nodes[0]
		html := NodeChildren2html(t3)
		rule := regexp.MustCompile(`<div class="pdsec">(.+?)</div>`)
		cnt := 0
		html = rule.ReplaceAllStringFunc(html, func(x string) string {
			cnt++
			match := rule.FindStringSubmatch(x)[1]
			return "\n# " + match + "\n\n"
		})
		if cnt == 0 {
			log.Println("Warning!")
			t5 := page.Find(`#pcont2`).Nodes[0]
			html2 := NodeChildren2html(t5)
			rule := regexp.MustCompile(`<p class="subtitle">(.+)</p>`)
			cnt := 0
			html2 = rule.ReplaceAllStringFunc(html2, func(x string) string {
				cnt++
				match := rule.FindStringSubmatch(x)[1]
				return "# " + match + "\n\n"
			})
			rule = regexp.MustCompile(`<[pb]>【(.+)】</[pb]>`)
			html2 = rule.ReplaceAllStringFunc(html2, func(x string) string {
				cnt++
				match := rule.FindStringSubmatch(x)[1]
				return "# " + match + "\n\n"
			})
			if cnt <= 0 || cnt >= 15 {
				log.Println("Error! cnt=", cnt)
				html = "# 题面\n\n"
				html += NodeChildren2html(t3)
			} else {
				html = html2
				log.Println(cnt)
			}
		} else {
			log.Println(cnt)
		}
		t4, err := DownloadImage(nil, html, homePath+p.Pid+"/img/", fileList, "http://www.tsinsen.com/"+p.Pid+"/", "http://www.tsinsen.com")
		if err != nil {
			log.Println(err)
		} else {
			html = t4
		}
		p.Data.Description = html
		pl = append(pl, p)
	}
	err := WriteFiles(pl, fileList, homePath)
	if err != nil {
		log.Panicln(err)
	}
	return fileList, nil
}

func Stop() {
	logger.Println("UniversalOJ crawler stopped")
}
