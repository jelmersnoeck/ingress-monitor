PKG=github.com/jelmersnoeck/ingress-monitor
PKGS := $(shell go list ./... | grep -v generated)

ci: bootstrap linters check-generated lint cover

#################################################
# Bootstrapping for base golang package deps
#################################################
BOOTSTRAP=github.com/golang/dep/cmd/dep

$(BOOTSTRAP):
	go get -u $@

bootstrap: $(BOOTSTRAP)

vendor: Gopkg.lock
	dep ensure -v -vendor-only

update-vendor:
	dep ensure -v -update

#################################################
# Testing and linting
#################################################
LINTER_PKGS=$(patsubst $(PKG)/%,./%,$(PKGS))
test: vendor
	CGO_ENABLED=0 go test -v ./...

linters:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

lint:
	golangci-lint run -D errcheck -D staticcheck ./...

cover: vendor
	CGO_ENABLED=0 go test -v -coverprofile=coverage.txt -covermode=atomic ./...

cover-html: vendor
	CGO_ENABLED=0 go test -coverprofile cover.out ./...
	go tool cover -html=cover.out -o cover.html
	open cover.html

#################################################
# Code Generation
#################################################
APIS=$(sort $(patsubst apis/%/,%,$(dir $(wildcard apis/*/*/))))

api-versions:
	@echo $(APIS)

$(APIS): vendor
	./vendor/k8s.io/code-generator/generate-groups.sh \
	  all \
	  $(PKG)/pkg/client/generated \
	  $(PKG)/apis \
	  $(subst /,:,$@) \
	  --go-header-file boilerplate.go.txt \
	  $@

clean-generated:
	rm -rf ./pkg/client/generated

generated: clean-generated $(APIS)

check-generated: generated
	@(git diff --exit-code . || (echo "Generated files are outdated" && exit 1))

#################################################
# Building binaries and docker images
#################################################
BASE_BRANCH=master
DOCKER_REPOSITORY=jelmersnoeck
GOOS_OVERRIDE?=
PREFIX?=

GO_BUILD=CGO_ENABLED=0 go build -i
DOCKER_MAKE=GOOS_OVERRIDE='GOOS=linux' PREFIX=build/docker/$1/ make build/docker/$1/bin/$1

CMDs=$(sort $(patsubst cmd/%/,%,$(dir $(wildcard cmd/*/))))
BINS=$(addprefix bin/,$(CMDs))
DOCKER_IMAGES=$(addprefix docker-,$(CMDs))
DOCKER_RELEASES=$(addprefix release-,$(CMDs))

VCS_SHA?=$(shell git rev-parse --verify HEAD)
BUILD_DATE?=$(shell git show -s --date=iso8601-strict --pretty=format:%cd $$VCS_SHA)
VCS_BRANCH?=$(shell git branch | grep \* | cut -f2 -d' ')


RELEASE_VERSION?=$(shell git describe --always --tags --dirty | sed 's/^v//')
ifdef TRAVIS_TAG
	RELEASE_VERSION=$(shell echo $(TRAVIS_TAG) | sed 's/^v//')
endif


RELEASE_NAME?=$(patsubst docker-%,%,$@)
ifdef TRAVIS_PULL_REQUEST_BRANCH
	RELEASE_VERSION=$(TRAVIS_PULL_REQUEST_SHA)
	RELEASE_NAME="$(patsubst docker-%,%,$@)-$(shell echo $(TRAVIS_PULL_REQUEST_BRANCH) | sed "s/[^[:alnum:].-]/-/g")"
	# Override VCS_BRANCH on travis because it uses the FETCH_HEAD
	VCS_BRANCH=$(TRAVIS_PULL_REQUEST_BRANCH)
endif

$(CMDs:%=build/docker/%/Dockerfile):
	mkdir -p $(@D)
	cp Dockerfile.dev $@

$(BINS:%=$(PREFIX)%): $(PREFIX)bin/%: vendor
	$(GOOS_OVERRIDE) $(GO_BUILD) -o $@ $(patsubst $(PREFIX)bin/%,./cmd/%/...,$@)
$(BINS:%=%-dev):
	$(call DOCKER_MAKE,$(patsubst bin/%-dev,%,$@))
bins: $(BINS:%=$(PREFIX)%)

$(DOCKER_IMAGES):
	docker build -t $(DOCKER_REPOSITORY)/$(patsubst docker-%,%,$@):latest \
		--label "org.label-schema.build-date"="$(BUILD_DATE)" \
		--label "org.label-schema.name"="$(RELEASE_NAME)" \
		--label "org.label-schema.vcs-ref"="$(VCS_SHA)" \
		--label "org.label-schema.vendor"="Siphoc" \
		--label "org.label-schema.version"="$(RELEASE_VERSION)" \
		--label "org.vcs-branch"="$(VCS_BRANCH)" \
		--build-arg BINARY=$(patsubst docker-%,bin/%,$@) \
		--build-arg PKG=$(PKG) \
		.
$(DOCKER_IMAGES:%=%-dev): docker-%-dev: build/docker/%/Dockerfile bin/%-dev
	docker build -t $(DOCKER_REPOSITORY)/$(patsubst docker-%-dev,%,$@):latest \
		--label "org.label-schema.build-date"="$(BUILD_DATE)" \
		--label "org.label-schema.name"="$(RELEASE_NAME)" \
		--label "org.label-schema.vcs-ref"="$(VCS_SHA)" \
		--label "org.label-schema.vendor"="Siphoc" \
		--label "org.label-schema.version"="$(RELEASE_VERSION)" \
		--label "org.vcs-branch"="$(VCS_BRANCH)" \
		--build-arg BINARY=bin/$(patsubst docker-%-dev,%,$@) \
		build/docker/$(patsubst docker-%-dev,%,$@)

docker: $(DOCKER_IMAGES)
docker-dev: $(DOCKER_IMAGES:%=%-dev)

docker-login:
	docker login -u="$$DOCKER_USERNAME" -p="$$DOCKER_PASSWORD"

$(DOCKER_RELEASES): release-%: docker-login docker-%
	docker tag $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@) $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@):$(RELEASE_VERSION)
	docker push $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@):$(RELEASE_VERSION)
ifeq ($(VCS_BRANCH),$(BASE_BRANCH))
	# On master, we want to push latest
	docker push $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@):latest
else
	# On branches, we want to push specific branch version and latest branch
	docker tag $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@) $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@):$(RELEASE_VERSION)
	docker push $(DOCKER_REPOSITORY)/$(patsubst release-%,%,$@):$(RELEASE_VERSION)
endif
release: $(DOCKER_RELEASES)

.PHONY: $(BINS:%=$(PREFIX)%) $(DOCKER_IMAGES) $(CMDs:%=build/docker/%/Dockerfile)
