# Zoraxy Dockerfile BEZ ZeroTieru
#
# Stejné jako oficiální docker/Dockerfile, ale vynechává kompilaci ZeroTier
# (ta slouží jen pro funkci "Global Area Network" a je to nejpomalejší část
# buildu). Pokud ZeroTier nepotřebuješ, tenhle build je výrazně rychlejší.
#
# Pozor: s tímhle image NEFUNGUJE ZEROTIER (env ZEROTIER musí zůstat "false").

## Build Zoraxy
FROM docker.io/golang:1.27.3-alpine3.24 AS build-zoraxy
RUN mkdir -p /opt/zoraxy/source/ && \
    mkdir -p /usr/local/bin/

COPY ./src/ /opt/zoraxy/source/

WORKDIR /opt/zoraxy/source/

RUN go mod tidy && \
    go build -buildvcs=false -o /usr/local/bin/zoraxy && \
    chmod 755 /usr/local/bin/zoraxy

## Build entrypoint
FROM docker.io/golang:1.27.3-alpine3.24 AS build-entrypoint

COPY ./entrypoint.go /opt/zoraxy/entrypoint/

WORKDIR /opt/zoraxy/entrypoint/

RUN go build -o /usr/local/bin/zoraxy-entrypoint entrypoint.go && \
    chmod 755 /usr/local/bin/zoraxy-entrypoint

## Main
FROM docker.io/alpine:3.24

RUN apk add --update --no-cache tzdata sudo netcat-openbsd libressl-dev openssh ca-certificates && \
    rm -rf /sbin/apk /var/cache/apk/* /tmp/*

COPY --from=build-zoraxy /usr/local/bin/zoraxy /usr/local/bin/zoraxy
COPY --from=build-entrypoint /usr/local/bin/zoraxy-entrypoint /usr/local/bin/zoraxy-entrypoint

RUN mkdir -p /opt/zoraxy/plugin/

WORKDIR /opt/zoraxy/config/

ENV TZ="Etc/UTC"
ENV ZEROTIER="false"

ENV AUTORENEW="86400"
ENV CFGUPGRADE="true"
ENV DB="auto"
ENV DOCKER="true"
ENV EARLYRENEW="30"
ENV ENABLELOG="true"
ENV FASTGEOIP="false"
ENV MDNS="true"
ENV MDNSNAME="''"
ENV NOAUTH="false"
ENV PLUGIN="/opt/zoraxy/plugin/"
ENV PORT="8000"
ENV SSHLB="false"
ENV UPDATE_GEOIP="false"
ENV VERSION="false"
ENV WEBROOT="./www"

VOLUME [ "/opt/zoraxy/config/" ]

LABEL com.imuslab.zoraxy.container-identifier="Zoraxy"

ENTRYPOINT [ "/usr/local/bin/zoraxy-entrypoint" ]

HEALTHCHECK --start-period=5s --interval=15s --timeout=5s --retries=3 CMD nc -vz 127.0.0.1 $PORT || exit 1
