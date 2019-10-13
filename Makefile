build: crawler plugin/uoj/uoj plugin/loj/loj plugin/seuoj/seuoj plugin/guoj/guoj
clean:
	rm crawler rpc/api.pb.go plugin/uoj/uoj plugin/loj/loj plugin/seuoj/seuoj plugin/guoj/guoj
crawler: main.go plugin/public/tools.go rpc/api.pb.go
	go build ./
rpc/api.pb.go: rpc/api.proto rpc/gen.go
	go generate rpc/gen.go
plugin/uoj/uoj: plugin/uoj/uoj.go plugin/public/tools.go rpc/api.pb.go
	go build -o ./plugin/uoj/uoj ./plugin/uoj/
plugin/loj/loj: plugin/loj/loj.go plugin/public/tools.go rpc/api.pb.go plugin/syzoj/main.go
	go build -o ./plugin/loj/loj ./plugin/loj
plugin/guoj/guoj: plugin/guoj/guoj.go plugin/public/tools.go rpc/api.pb.go plugin/syzoj/main.go
	go build -o ./plugin/guoj/guoj ./plugin/guoj
plugin/seuoj/seuoj: plugin/seuoj/seuoj.go plugin/public/tools.go rpc/api.pb.go plugin/syzoj/main.go
	go build -o ./plugin/seuoj/seuoj ./plugin/seuoj
.PHONY: build
.IGNORE: clean
