EXECUTABLE=device-monitor
LOG_FILE=/var/log/${EXECUTABLE}.log
GOFMT=gofmt -w
GODEPS=go get -u

build:
	go build -o ${EXECUTABLE}

install:
	go install

format:
	${GOFMT} main.go
	${GOFMT} utils/adb_utils.go

test:

deps:
	${GODEPS} github.com/pborman/uuid
	${GODEPS} github.com/yosemite-open/go-adb
	${GODEPS} golang.org/x/net/websocket

stop:
	pkill -f ${EXECUTABLE}

start:
	-make stop
	cd ${GOPATH}/src/github.com/prsolucoes/${EXECUTABLE}
	nohup ${EXECUTABLE} >> ${LOG_FILE} 2>&1 </dev/null &

update:
	git pull origin master
	make install

build-all:
	rm -rf build

	mkdir -p build/linux32
	env GOOS=linux GOARCH=386 go build -o build/linux32/${EXECUTABLE} -v github.com/prsolucoes/${EXECUTABLE}

	mkdir -p build/linux64
	env GOOS=linux GOARCH=amd64 go build -o build/linux64/${EXECUTABLE} -v github.com/prsolucoes/${EXECUTABLE}

	mkdir -p build/darwin64
	env GOOS=darwin GOARCH=amd64 go build -o build/darwin64/${EXECUTABLE} -v github.com/prsolucoes/${EXECUTABLE}

	mkdir -p build/windows32
	env GOOS=windows GOARCH=386 go build -o build/windows32/${EXECUTABLE} -v github.com/prsolucoes/${EXECUTABLE}

	mkdir -p build/windows64
	env GOOS=windows GOARCH=amd64 go build -o build/windows64/${EXECUTABLE} -v github.com/prsolucoes/${EXECUTABLE}