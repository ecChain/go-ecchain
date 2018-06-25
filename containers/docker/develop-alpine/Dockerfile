FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/ecchain/go-ecchain && \
  (cd go-ecereum && make gec) && \
  cp go-ecereum/build/bin/gec /gec && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-ecereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/gec"]
