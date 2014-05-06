# to build windows version

.PHONY:
all: build upload

.PHONY:
build:
	ssh Admin@192.168.1.100 build.bat
	scp win:'C:\\go-workspace\\bin\\life-goes-on.exe' life-goes-on.exe

.PHONY:
upload:
	fshare life-goes-on.exe
