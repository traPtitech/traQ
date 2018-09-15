FROM golang:1.11.0-alpine AS build
RUN apk add --update --no-cache git
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/traPtitech/traQ
COPY ./Gopkg.* ./
RUN dep ensure --vendor-only=true
COPY . .
RUN go build -o /traQ


FROM alpine:3.8
WORKDIR /app

RUN apk add --update ca-certificates imagemagick openssl && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

VOLUME /localstorage
EXPOSE 80
ENV TRAQ_PORT=80 \
    TRAQ_ORIGIN=http://localhost \
    IMAGEMAGICK_EXEC=/usr/bin/convert \
    TRAQ_LOCAL_STORAGE=/localstorage

COPY ./static ./static/
COPY --from=build /traQ ./

ENTRYPOINT ./traQ
