# to build windows version

.PHONY:
all: win-build win-upload win-build-client win-upload-client

.PHONY:
win-build:
	ssh Admin@192.168.1.100 build.bat
	scp win:'C:\\go-workspace\\bin\\life-goes-on.exe' life-goes-on.exe

.PHONY:
win-upload:
	fshare life-goes-on.exe

.PHONY:
win-build-client:
	ssh Admin@192.168.1.100 build-client.bat
	scp win:'C:\\go-workspace\\bin\\life-goes-on.exe' life-goes-on-client.exe

.PHONY:
win-upload-client:
	fshare life-goes-on-client.exe

.PHONY:
build:
	go build

.PHONY:
server-update: build
	scp life-goes-on satelles:
	ssh satelles ./update-lgo

.PHONY:
sup: server-update
