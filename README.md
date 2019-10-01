# OI-Archive 题库爬虫

本项目为 OI-Archive 的题库爬虫。



 ## 编译运行

```shell
sudo apt install golang-go
go get github.com/oi-archive/crawler
cd ~/go/src/github.com/oi-archive/crawler
make
./crawler
```



## 插件 API

每个题库的爬虫都以插件的形式被爬虫主程序调用，具体的格式如下

#### Go

对于 go 语言，需要实现以下接口，然后以 plugin 模式编译，即可正常被主程序调用。

```go
func Name() string // 返回题库名称
func Start(logg *log.Logger)  // 在且仅在插件初始化时被调用一次

// 每次主程序要求爬虫进行一次更新时会被调用
// limit: 主程序希望爬虫这一次爬取的题目数量（非严格要求，爬虫可以自行决定到底爬几题）
// public.Filelist: map[string][]byte 类型，表示这次更新的文件列表，key表示文件的完整路径名，value表示文件内容
// error: 本次爬虫运行是否出现致命错误。若非空则主程序将忽略这次爬取的结果。
func Update(limit int) (public.FileList, error) 

func Stop() // 可能在插件关闭时被调用
```



#### 其他语言

对于其他语言，你只需要用你喜欢的方式实现上面的几个接口，然后导出为 C 语言格式的库文件即可。