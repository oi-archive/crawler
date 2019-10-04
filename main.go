package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/oi-archive/crawler/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/libgit2/git2go.v26"
	"io/ioutil"
	"log"
	"net"
	"plugin"
	"sync"
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

var gitMutex sync.Mutex

func addFileAndCommit(fileList map[string][]byte, problemsetName string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()
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
	log.Println(commitID)
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
	gitMutex.Lock()
	defer gitMutex.Unlock()
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

type server struct{}

type Plugin struct {
	id   string
	name string
}

var plu map[string]*Plugin

func (s *server) Register(c context.Context, req *rpc.RegisterRequest) (*rpc.RegisterReply, error) {
	log.Println(req.Id, req.Name)
	plu[req.Id] = &Plugin{id: req.Id, name: req.Name}
	return &rpc.RegisterReply{DebugMode: true}, nil
}

func (s *server) Deregister(c context.Context, req *rpc.DeregisterRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (s *server) Update(c context.Context, req *rpc.UpdateRequest) (*rpc.UpdateReply, error) {
	err := addFileAndCommit(req.File, req.Id)
	if err != nil {
		log.Println("git error:", err)
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
		return &rpc.UpdateReply{Ok: false}, nil
	} else {
		err = gitPush()
		if err != nil {
			log.Println("git push error:", err)
			return &rpc.UpdateReply{Ok: false}, nil
		}
	}
	return &rpc.UpdateReply{Ok: true}, nil
}

func main() {
	var err error
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
	plu = make(map[string]*Plugin)
	lis, err := net.Listen("tcp", ":27381")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	rpc.RegisterAPIServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
