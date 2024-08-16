FROM golang:1.22.6-alpine AS build

WORKDIR /build
COPY . .
RUN unset GOPATH && \
    mkdir dist && \
    go build -tags musl -o ./dist ./...

FROM alpine:latest AS release
RUN adduser -D tidepool
WORKDIR /home/tidepool
USER tidepool
COPY --chown=tidepool --from=build /build/dist/mailer ./mailer
CMD ["./mailer"]
