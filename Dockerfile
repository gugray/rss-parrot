FROM alpine:latest

WORKDIR /app
COPY bin /app/
RUN chmod +x ./rss_parrot
ENTRYPOINT [ "./rss_parrot" ]
