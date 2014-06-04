# to build windows version

.PHONY:
all: win-build win-upload

.PHONY:
win-build:
	ssh Admin@192.168.1.100 build.bat
	scp win:'C:\\go-workspace\\bin\\life-goes-on.exe' life-goes-on.exe

.PHONY:
win-upload:
	fshare life-goes-on.exe

.PHONY:
build:
	go build

.PHONY:
server-update: build
	scp life-goes-on satelles:
	ssh satelles ./update-lgo

.PHONY:
sup: server-update
