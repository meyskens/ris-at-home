ARG BUILDPLATFORM="linux/amd64"
FROM --platform=$BUILDPLATFORM golang:1.22-alpine as build

ARG TARGETPLATFORM
ARG BUILDPLATFORM

RUN apk add --no-cache git

COPY ./ /go/src/github.com/meyskens/ris-at-home

WORKDIR /go/src/github.com/meyskens/ris-at-home

RUN export GOARM=6 && \
    export GOARCH=amd64 && \
    if [ "$TARGETPLATFORM" == "linux/arm64" ]; then export GOARCH=arm64; fi && \
    if [ "$TARGETPLATFORM" == "linux/arm" ]; then export GOARCH=arm; fi && \
    go build -ldflags "-X main.revision=$(git rev-parse --short HEAD)" ./apiserver/cmd/risapi/

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

RUN mkdir -p /go/src/github.com/meyskens/ris-at-home
WORKDIR /go/src/github.com/meyskens/ris-at-home

COPY --from=build /go/src/github.com/meyskens/ris-at-home/risapi /usr/local/bin/
COPY --from=build /go/src/github.com/meyskens/ris-at-home/public /go/src/github.com/meyskens/ris-at-home/public

ENTRYPOINT [ "/usr/local/bin/risapi" ]
CMD [ "serve" ]