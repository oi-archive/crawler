package main

import (
	"encoding/json"
	"fmt"
	"github.com/robfig/cron"
	"gopkg.in/libgit2/git2go.v26"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"time"
)

var P []*plugin.Plugin
var gitRepo *git.Repository

type Sshkey struct {
	Public_key  string
	Private_key string
}

var sshkey Sshkey

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

func gitPush() error {
	remote, err := gitRepo.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	err = remote.Push([]string{"refs/heads/master"}, &git.PushOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback: func(url string, username_from_url string, allowed_types git.CredType) (git.ErrorCode, *git.Cred) {
				ret, cred := git.NewCredSshKey("git", sshkey.Public_key, sshkey.Private_key, "")
				return git.ErrorCode(ret), &cred
			},
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				// 忽略服务端证书错误
				return git.ErrorCode(0)
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}
func runUpdate() {
	for _, p := range P {
		pName := try(p.Lookup("Name")).(func() string)()
		fileList, err := try(p.Lookup("Update")).(func(int) (map[string][]byte, error))(200)
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
		} else {
			err = gitPush()
			if err != nil {
				log.Println("git push error:", err)
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
	log.Println("插件载入完成")
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
	b, err := ioutil.ReadFile("config/sshkey.json")
	if err != nil {
		log.Panicln(err)
	}
	err = json.Unmarshal(b, &sshkey)
	if err != nil {
		log.Panicln(err)
	}
	runUpdate()
	c := cron.New()
	_ = c.AddFunc("@midnight", runUpdate)
	c.Start()
	select {}
}
