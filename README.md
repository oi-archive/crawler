# OI-Archive 题库爬虫

本项目为 OI-Archive 的题库爬虫。

项目分为一个主服务和若干组件，每个组件负责一个题库。主服务和组件间使用 grpc 连接。



 ## 编译

首先安装 protobuf

```shell
sudo apt install golang-go
go get github.com/oi-archive/crawler
cd ~/go/src/github.com/oi-archive/crawler
make
```



### 运行

* 启动主服务 `./crawler`
* 分别运行 `plugin` 目录中的所有组件



## 开发指南

主服务提供的 API 见 `rpc/api.proto` （相信大家都能看懂 protobuf 文件，即使看不懂也没关系，可以看下面的各语言示例）

#### Go 

把 `plugin/example-go`复制一份，然后在标记了 `TODO: ` 的位置编写你的代码。

### Python3

环境准备：

```shell
pip3 install grpcio
pip3 install grpcio-tools
pip3 install apscheduler
```

把 `plugin/example-python`复制一份，进入新的目录

```shell
python3 -m grpc_tools.protoc -I../../rpc/ --python_out=. --grpc_python_out=. ../../rpc/api.proto
```

然后在标记了 `TODO: ` 的位置编写你的代码。

### 其他语言

如果需要用其他语言开发爬虫，请联系 @WAAutoMaton 获取技术支持。