build: crawler plugin/loj.so plugin/bzoj.so plugin/uoj.so plugin/guoj.so plugin/cogs.so plugin/seuoj.so
clean:
	rm plugin/*.so
crawler: main.go plugin/public/tools.go
	go build ./
plugin/loj.so: plugin/loj/loj.go plugin/public/tools.go plugin/syzoj/main.go
	go build -buildmode=plugin -o ./plugin/loj.so  ./plugin/loj/
plugin/seuoj.so: plugin/seuoj/seuoj.go plugin/public/tools.go plugin/syzoj/main.go
	go build -buildmode=plugin -o ./plugin/seuoj.so  ./plugin/seuoj/
plugin/bzoj.so: plugin/bzoj/bzoj.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/bzoj.so  ./plugin/bzoj/
plugin/uoj.so: plugin/uoj/uoj.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/uoj.so  ./plugin/uoj/
plugin/guoj.so: plugin/guoj/guoj.go plugin/public/tools.go plugin/syzoj/main.go
	go build -buildmode=plugin -o ./plugin/guoj.so  ./plugin/guoj/
plugin/tsinsen.so: plugin/tsinsen/tsinsen.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/tsinsen.so  ./plugin/tsinsen/
plugin/cogs.so: plugin/cogs/cogs.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/cogs.so  ./plugin/cogs/

.PHONY: build
.IGNORE: clean
