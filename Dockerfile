FROM alpine:latest

WORKDIR /app
COPY bin /app/
ENTRYPOINT [ "./rss_parrot" ]
