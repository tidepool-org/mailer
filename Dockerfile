FROM alpine:latest
RUN adduser -D tidepool
WORKDIR /home/tidepool
USER tidepool
COPY --chown=tidepool ./dist/mailer_linux ./mailer
CMD ["./mailer"]
