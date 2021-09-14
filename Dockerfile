FROM golang:1.17-alpine AS build

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
