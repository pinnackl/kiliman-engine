default:
	make setup
	make build
	make test

setup:
	go get -t github.com/onsi/ginkgo
	go install github.com/onsi/ginkgo/ginkgo
	go get -u github.com/golang/dep
	go install github.com/golang/dep/cmd/dep

build:
	dep ensure
	rm -rf vendor
	go build -i

clean:
	go clean

test:
	ginkgo -r

beforecommit:
	make fmt
	make test
	make clean

