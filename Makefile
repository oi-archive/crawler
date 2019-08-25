build: crawler plugin/libreoj.so plugin/bzoj.so
crawler: main.go
	go build ./
plugin/libreoj.so: plugin/libreoj/main.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/libreoj.so  ./plugin/libreoj/
plugin/bzoj.so: plugin/bzoj/main.go plugin/public/tools.go
	go build -buildmode=plugin -o ./plugin/bzoj.so  ./plugin/bzoj/

.PHONY: build

