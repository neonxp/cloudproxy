FROM golang:1.13 as builder
WORKDIR /usr/src
COPY go.mod .
COPY go.sum .
RUN GOPROXY=${PROXY} go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxy .

FROM alpine
RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
        && update-ca-certificates 2>/dev/null || true
WORKDIR /usr/app
COPY --from=builder /usr/src/proxy .
CMD ["./proxy"]