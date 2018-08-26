PKG=github.com/jelmersnoeck/ingress-monitor
PKGS := $(shell go list ./... | grep -v generated)

ci: check-generated lint test

#################################################
# Bootstrapping for base golang package deps
#################################################
BOOTSTRAP=\
	github.com/golang/dep/cmd/dep \
	github.com/alecthomas/gometalinter \

$(BOOTSTRAP):
	go get -u $@

bootstrap: $(BOOTSTRAP)
	gometalinter --install

vendor: Gopkg.lock
	dep ensure -v -vendor-only

update-vendor:
	dep ensure -v -update

#################################################
# Testing and linting
#################################################
test: vendor
	CGO_ENABLED=0 go test -v ./...

cover: vendor
	CGO_ENABLED=0 go test -v -coverprofile=coverage.txt -covermode=atomic ./...

cover-html: vendor
	CGO_ENABLED=0 go test -coverprofile cover.out ./...
	go tool cover -html=cover.out -o cover.html
	open cover.html

lint:
	gometalinter --tests --enable-all --vendor --deadline=5m -e "zz_.*\.go" $(PKGS)

#################################################
# Code Generation
#################################################
APIS=$(sort $(patsubst apis/%/,%,$(dir $(wildcard apis/*/*/))))

api-versions:
	@echo $(APIS)

$(APIS):
	./vendor/k8s.io/code-generator/generate-groups.sh \
	  all \
	  $(PKG)/pkg/client/generated \
	  $(PKG)/apis \
	  $(subst /,:,$@) \
	  --go-header-file boilerplate.go.txt \
	  $@

generated: $(APIS)

check-generated: generated
	@(git diff --exit-code . || (echo "Generated files are outdated" && exit 1))
