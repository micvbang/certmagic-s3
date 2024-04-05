#####################################
############ BUILDER ################
#####################################
FROM caddy:2.7-builder-alpine AS builder

ARG GOOS="linux"
ARG GOARCH="amd64"

RUN GOOS=${GOOS} GOARCH=${GOARCH} xcaddy build \
    --with github.com/micvbang/certmagic-s3@v0.0.3

#####################################
############ RUNNER #################
#####################################
FROM caddy:2.7-alpine as runner

COPY --from=builder /usr/bin/caddy /usr/bin/caddy