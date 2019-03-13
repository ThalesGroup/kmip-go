FROM golang:1.11.5-alpine

# ENV http_proxy=$http_proxy
# ENV http_proxy=http://$HTTP_PROXY
RUN apk --no-cache add make git curl bash fish

# build tools
COPY ./Makefile /go/src/github.com/gemalto/kmip-go/
WORKDIR /go/src/github.com/gemalto/kmip-go
RUN make tools

COPY ./ /go/src/github.com/gemalto/kmip-go

CMD make all

# unset proxy vars, just used in build
# ENV http_proxy=
