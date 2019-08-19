package main

import (
	"fmt"
	"github.com/robfig/cron"
	"gopkg.in/libgit2/git2go.v26"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"time"
)

var P []*plugin.Plugin
var gitRepo *git.Repository

func try(x interface{}, err error) interface{} {
	return x
}

func addFileAndCommit(fileList map[string][]byte, problemsetName string) error {
	sig := &git.Signature{
		Name:  "OI-Archive Crawler",
		Email: "null",
		When:  time.Now(),
	}
	index, err := gitRepo.Index()
	if err != nil {
		return err
	}

	for path, file := range fileList {
		oid, err := gitRepo.CreateBlobFromBuffer(file)
		if err != nil {
			return err
		}
		ie := git.IndexEntry{
			Mode: git.FilemodeBlob,
			Id:   oid,
			Path: path,
		}
		err = index.Add(&ie)
		if err != nil {
			return err
		}
	}
	treeID, err := index.WriteTree()
	if err != nil {
		return err
	}
	tree, err := gitRepo.LookupTree(treeID)
	if err != nil {
		return err
	}
	currentBranch, err := gitRepo.Head()
	if err != nil {
		return err
	}
	currentTip, err := gitRepo.LookupCommit(currentBranch.Target())
	if err != nil {
		return err
	}
	commitID, err := gitRepo.CreateCommit("HEAD", sig, sig, fmt.Sprintf("Problemset %s updated:%s", problemsetName, time.Now().String()), tree, currentTip)
	if err != nil {
		return err
	}
	fmt.Println(commitID)
	nextTip, err := gitRepo.LookupCommit(commitID)
	if err != nil {
		return err
	}
	err = gitRepo.ResetToCommit(nextTip, git.ResetHard, &git.CheckoutOpts{})
	if err != nil {
		return err
	}
	return nil
}

func runUpdate() {
	for _, p := range P {
		pName := try(p.Lookup("Name")).(func() string)()
		fileList, err := try(p.Lookup("Update")).(func(int) (map[string][]byte, error))(114514)
		if err != nil {
			log.Printf(`call "Update" error in plugin %s: %v\n`, pName, err)
			continue
		}
		err = addFileAndCommit(fileList, pName)
		if err != nil {
			log.Println("git err:", err)
			currentBranch, err := gitRepo.Head()
			if err != nil {
				log.Panicln("git error:", err)
			}
			currentTip, err := gitRepo.LookupCommit(currentBranch.Target())
			if err != nil {
				log.Panicln("git error:", err)
			}
			err = gitRepo.ResetToCommit(currentTip, git.ResetHard, &git.CheckoutOpts{})
			if err != nil {
				log.Panicln("git error:", err)
			}
		}
		log.Println("Updated " + pName)
	}
}
func main() {
	err := filepath.Walk("plugin", func(path string, info os.FileInfo, err error) error {
		// 遍历目录查找插件
		if info.IsDir() {
			return nil
		}
		p, err := plugin.Open(path)
		if err != nil {
			return nil
		}
		// 插件接口检查
		f, err := p.Lookup("Name")
		if err != nil {
			log.Panicln(`Lookup "Name" in plugin %s error`, path)
		}
		if _, ok := f.(func() string); !ok {
			log.Panicln(`Check "Name" in plugin %s error`, path)
		}
		f, err = p.Lookup("Start")
		if err != nil {
			log.Panicln(`Lookup "Start" in plugin %s error`, path)
		}
		if _, ok := f.(func() error); !ok {
			log.Panicln(`Check "Start" in plugin %s error`, path)
		}
		f, err = p.Lookup("Update")
		if err != nil {
			log.Panicln(`Lookup "Update" in plugin %s error`, path)
		}
		if _, ok := f.(func(int) (map[string][]byte, error)); !ok {
			log.Panicln(`Check "Update" in plugin %s error`, path)
		}
		f, err = p.Lookup("Stop")
		if err != nil {
			log.Panicln(`Lookup "Stop" in plugin %s error`, path)
		}
		if _, ok := f.(func()); !ok {
			log.Panicln(`Check "Stop" in plugin %s error`, path)
		}
		log.Printf("open plguin %s succeed", path)
		P = append(P, p)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	for _, p := range P {
		err := try(p.Lookup("Start")).(func() error)()
		if err != nil {
			log.Panicf(`call "Start" error in plugin %s: %v\n`, try(p.Lookup("Name")).(func() string)(), err)
		}
	}
	gitRepo, err = git.OpenRepository("../source")
	if err != nil {
		log.Panicln(err)
	}

	runUpdate()
	c := cron.New()
	_ = c.AddFunc("@midnight", runUpdate)
	c.Start()
	select {}
}
