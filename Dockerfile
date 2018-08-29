FROM golang:1.11

ARG BINARY
ARG PKG

WORKDIR /go/src/$PKG
COPY . ./

RUN make bootstrap
RUN make vendor
RUN make $BINARY
RUN cp $BINARY /operator

FROM manifoldco/scratch-certificates
USER 7171:8787

COPY --from=0 /operator /operator
ENTRYPOINT ["/operator"]
