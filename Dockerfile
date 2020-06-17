FROM unitedwardrobe/golang-librdkafka:alpine3.11-golang1.14.3-librdkafka1.4.2-static AS build

WORKDIR /build
COPY . .
RUN unset GOPATH && \
    go build -tags musl -o ./dist ./...

FROM alpine:latest AS release
RUN adduser -D tidepool
WORKDIR /home/tidepool
USER tidepool
COPY --chown=tidepool --from=build /build/dist/mailer ./mailer
CMD ["./mailer"]
