build: crawler plugin/loj.so plugin/bzoj.so plugin/uoj.so plugin/guoj.so
crawler: main.go plugin/public/tools.go
	go build ./
plugin/loj.so: plugin/loj/main.go plugin/public/tools.go plugin/syzoj/main.go
	go build -buildmode=plugin -o ./plugin/loj.so  ./plugin/loj/
plugin/bzoj.so: plugin/bzoj/main.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/bzoj.so  ./plugin/bzoj/
plugin/uoj.so: plugin/uoj/main.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/uoj.so  ./plugin/uoj/
plugin/guoj.so: plugin/guoj/main.go plugin/public/tools.go plugin/syzoj/main.go
	go build -buildmode=plugin -o ./plugin/guoj.so  ./plugin/guoj/

.PHONY: build

