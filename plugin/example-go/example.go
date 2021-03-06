package main

import (
	"context"
	. "crawler/plugin/public"
	"crawler/rpc"
	"google.golang.org/grpc"
	"log"
)

var client rpc.APIClient

const PID = "example"     //TODO: 题库代号
const NAME = "Example OJ" //TODO: 题库全名

var info *rpc.Info

var fileList map[string][]byte

var debugMode bool

// 该组件启动时被调用一次
// TODO: 在此方法中编写初始化代码
func Start() error {
	return nil
}

// 每次更新时被调用
// FileList: map[string][]byte,表示此次要提交更新的文件列表，key表示文件完整路径名，value表示文件内容
// TODO: 在此方法中编写爬虫程序
func Update() (FileList, error) {
	return fileList, nil
}

// 组件结束运行时被调用
// TODO: 在此方法中执行释放资源等任务
func Stop() {
}

func runUpdate() {
	file, err := Update()
	if err != nil {
		log.Println("Update Error")
		return
	}
	r, err := client.Update(context.Background(), &rpc.UpdateRequest{Info: info, File: file})
	if err != nil {
		log.Printf("Submit update failed: %v", err)
	}
	if !r.Ok {
		log.Println("Submit update failed")
	}
}

func main() {
	info = &rpc.Info{Id: PID, Name: NAME}
	conn, err := grpc.Dial("127.0.0.1:27381", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client = rpc.NewAPIClient(conn)
	err = Start()
	if err != nil {
		log.Panicln(err)
	}
	r, err := client.Register(context.Background(), &rpc.RegisterRequest{Info: info})
	if err != nil {
		log.Fatalf("could not register: %v", err)
	}
	log.Println(r.DebugMode)
	debugMode = r.DebugMode
	runUpdate()
}
