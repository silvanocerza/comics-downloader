# get latest git tag
TAG = $$(git describe --abbrev=0 --tags)
# get current date and time
NOW = $$(date +'%Y-%m-%d_%T')
# use the linker to inject the informations
LDFLAGS = "-X main.release=${TAG} -X main.buildTime=${NOW}"
LDFLAGSWINDOWS = "-H windowsgui -X main.release=${TAG} -X main.buildTime=${NOW}"

help:                              # this command
	# [generating help from tasks header]
	@egrep '^[A-Za-z0-9_-]+:' Makefile

osx-build: # Creates Mac OSX
	@GOOS=darwin GOARCH=amd64 go build -ldflags=${LDFLAGS} -o build/comics-downloader-osx ./cmd/downloader

windows-build: # Creates Windows
	@GOOS=windows GOARCH=amd64 go build -ldflags=${LDFLAGS}  -o build/comics-downloader.exe ./cmd/downloader

linux-build: # Creates Linux
	@GOOS=linux GOARCH=amd64 go build -ldflags=${LDFLAGS} -o build/comics-downloader ./cmd/downloader

linux-arm-build: # Creates Linux ARM
	@GOOS=linux GOARCH=arm go build -ldflags=${LDFLAGS} -o build/comics-downloader-linux-arm ./cmd/downloader

linux-arm64-build: # Creates Linux ARM64
	@GOOS=linux GOARCH=arm64 go build -ldflags=${LDFLAGS} -o build/comics-downloader-linux-arm64 ./cmd/downloader

osx-gui-build: # Creates OSX Gui
	@GOOS=darwin GOARCH=amd64 go build -ldflags=${LDFLAGS} -o build/comics-downloader-gui-osx ./cmd/gui

windows-gui-build: # Creates Window GUI executable
	@CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc go build -ldflags=${LDFLAGSWINDOWS} -o build/comics-downloader-gui-windows.exe ./cmd/gui

linux-gui-build: # Creates LINUX executable
	@fyne-cross --ldflags=${LDFLAGS} --output=comics-downloader-gui --targets=linux/amd64 ./cmd/gui

builds: # Creates executables for OSX/Windows/Linux
	@make osx-build
	@make windows-build
	@make linux-build
	@make windows-gui-build
	@make osx-gui-build
	@make linux-gui-build
	@make linux-arm-build
	@make linux-arm64-build

remove-builds: # Remove executables
	@rm -rf build/
