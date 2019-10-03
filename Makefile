build: crawler plugin/uoj/uoj
clean:
	rm crawler rpc/api.pb.go plugin/uoj/uoj
crawler: main.go plugin/public/tools.go rpc/api.pb.go
	go build ./
rpc/api.pb.go: rpc/api.proto rpc/gen.go
	go generate rpc/gen.go
plugin/uoj/uoj: plugin/uoj/uoj.go plugin/public/tools.go rpc/api.pb.go
	go build -o ./plugin/uoj/uoj ./plugin/uoj/
.PHONY: build
.IGNORE: clean
