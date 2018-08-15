LDFLAGS = "-w -s -X main.BUILD_DATE=`date '+%Y-%m-%d_%I:%M:%S'` -X main.GitHash=`git rev-parse HEAD` -X main.VERSION=`git describe --tag --always`"
BUILD_PATH=bin
BUILD_NAME = supergo
CTL_BUILD_NAME = supergoctl
.PHONY:clean build run tar vet

default:build

build:vet
	@go build  -ldflags ${LDFLAGS} -o ${BUILD_PATH}/${BUILD_NAME} ./cmd/supergo
	@go build  -ldflags ${LDFLAGS} -o ${BUILD_PATH}/${CTL_BUILD_NAME} ./cmd/supergoctl

tar:build
	tar -czvf ${BUILD_NAME}.tar.gz bin config

clean:
	@rm -rf log ${BUILD_NAME} ${CTL_BUILD_NAME}

run:build
	./${BUILD_NAME}

vet:
	@go vet ./...