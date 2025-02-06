FROM alpine:latest AS build
RUN apk update
RUN apk upgrade
RUN apk add --update gcc g++
RUN apk add --no-cache go=go-1.23.5

WORKDIR /repo
COPY . /repo
RUN sh ./build.sh

FROM alpine:latest

WORKDIR /app
COPY --from=build repo/bin /app/
RUN chmod +x ./rss_parrot
ENTRYPOINT [ "./rss_parrot" ]
