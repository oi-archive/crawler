package main

import (
	"github.com/robfig/cron"
	"gopkg.in/libgit2/git2go.v26"
	"log"
	"os"
	"path/filepath"
	"plugin"
)

var P []*plugin.Plugin
var gitRepo *git.Repository
var gitOdb *git.Odb
func runUpdate() {
	for _,p:=range P {
		update,err:=p.Lookup("Update")
		if err!=nil {
			log.Panicln(`lookup "Update" error`)
		}
		if update,ok:=update.(func() error); ok {
			err:=update()
			if err!=nil {
				log.Println(`call "Update" error:`,err)
			}
		} else {
			log.Panicln(`call "Update" error`)
		}
	}
}
func main() {
	err:=filepath.Walk("plugin",func(path string,info os.FileInfo,err error) error{
		if info.IsDir() {
			return nil
		}
		p,err:=plugin.Open(path)
		if err!=nil {
			return nil
		}
		log.Printf("open plguin %s succeed",path)
		P=append(P,p)
		return nil
	})
	if err!=nil {
		log.Panic(err)
	}
	for _,p:=range P {
		start,err:=p.Lookup("Start")
		if err!=nil {
			log.Panicln(`lookup "Start" error`)
		}
		if start,ok:=start.(func() error); ok {
			err:=start()
			if err!=nil {
				log.Panicln(`call "Start" error:`,err)
			}
		} else {
			log.Panicln(`call "Start" error`)
		}
	}
	gitRepo,err=git.OpenRepository("../source")
	if err!=nil {
		log.Panicln(err)
	}
	gitOdb,err=gitRepo.Odb()

	runUpdate()
	c:=cron.New()
	_ = c.AddFunc("@midnight", runUpdate)
	c.Start()
	select {

	}
	for _,p:=range P {
		stop,err:=p.Lookup("Stop")
		if err!=nil {
			log.Panicln(`lookup "Stop" error`)
		}
		if stop,ok:=stop.(func()); ok {
			stop()
		} else {
			log.Panicln(`call "Stop" error`)
		}
	}
}
