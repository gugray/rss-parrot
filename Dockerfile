FROM alpine:latest

RUN apk add --no-cache git make build-base

WORKDIR /app
COPY bin /app/
RUN chmod +x ./rss_parrot
ENTRYPOINT [ "./rss_parrot" ]
