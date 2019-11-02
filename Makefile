build: crawler plugin/uoj/uoj plugin/loj/loj plugin/seuoj/seuoj plugin/guoj/guoj plugin/bzoj/bzoj plugin/lutece/lutece plugin/joyoi/joyoi
clean:
	rm crawler rpc/api.pb.go plugin/uoj/uoj plugin/loj/loj plugin/seuoj/seuoj plugin/guoj/guoj plugin/bzoj/bzoj plugin/lutece/lutece plugin/joyoi/joyoi
crawler: main.go plugin/public/tools.go rpc/api.pb.go
	go build ./
rpc/api.pb.go: rpc/api.proto rpc/gen.go
	go generate rpc/gen.go
plugin/uoj/uoj: plugin/uoj/uoj.go plugin/public/tools.go rpc/api.pb.go
	go build -o ./plugin/uoj/uoj ./plugin/uoj/
plugin/loj/loj: plugin/loj/loj.go plugin/public/tools.go rpc/api.pb.go plugin/syzoj/main.go
	go build -o ./plugin/loj/loj ./plugin/loj/
plugin/guoj/guoj: plugin/guoj/guoj.go plugin/public/tools.go rpc/api.pb.go plugin/syzoj/main.go
	go build -o ./plugin/guoj/guoj ./plugin/guoj/
plugin/seuoj/seuoj: plugin/seuoj/seuoj.go plugin/public/tools.go rpc/api.pb.go plugin/syzoj/main.go
	go build -o ./plugin/seuoj/seuoj ./plugin/seuoj/
plugin/bzoj/bzoj: plugin/bzoj/bzoj.go plugin/public/tools.go rpc/api.pb.go
	go build -o ./plugin/bzoj/bzoj ./plugin/bzoj/
plugin/lutece/lutece: plugin/lutece/lutece.go plugin/public/tools.go rpc/api.pb.go
	go build -o ./plugin/lutece/lutece ./plugin/lutece/
plugin/joyoi/joyoi: plugin/joyoi/joyoi.go plugin/public/tools.go rpc/api.pb.go
	go build -o ./plugin/joyoi/joyoi ./plugin/joyoi/
.PHONY: build
.IGNORE: clean
