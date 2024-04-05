# CertMagic-S3

CertMagic S3-compatible driver written in Go.

The driver utilizes the github.com/aws/aws-sdk-go/aws package to access the S3 API. Configuration of AWS credentials follows [AWS's standard environment variables, configuration files etc.](https://docs.aws.amazon.com/sdkref/latest/guide/creds-config-files.html). The precedence is [described here](https://docs.aws.amazon.com/sdk-for-go/api/aws/session/#hdr-Credential_and_config_loading_order).

## Build and run

Build

```bash
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

xcaddy build --output ./caddy --with github.com/micvbang/certmagic-s3
```

Build container

```Dockerfile
FROM caddy:builder AS builder
RUN xcaddy build --with github.com/micvbang/certmagic-s3 --with ...

FROM caddy
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

Run

```bash
caddy run --config caddy.json
```

## Configuration

Caddyfile Example

```json
{
    storage s3 {
        bucket "Bucket"
    }
}
```

JSON Config Example

```json
{
    "storage": {
        "bucket": "Bucket",
    }
}
```

From Environment

```bash
export S3_CERTIFICATE_BUCKET="bucket-name"
```
