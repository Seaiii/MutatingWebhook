build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-v GOGCFLAGS="all=-N -l" go build -ldflags "-X main.buildstamp=`date  '+%Y-%m-%d_%H:%M:%S'` -X 'main.goversion=$(go version)'" -mod vendor -o node main.go;
tag:= webhook-example:$(shell date "+%Y%m%d-%H%M%S")
image:
	make build;
	docker build . -t $(tag);