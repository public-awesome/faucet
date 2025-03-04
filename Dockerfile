FROM alpine:3.20
RUN apk add --no-cache curl ca-certificates

COPY ./bin/faucet-linux /usr/bin/faucet-server

ENTRYPOINT ["faucet-server"]

