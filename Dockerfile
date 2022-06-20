FROM golang:1.18 as builder
WORKDIR /data/
COPY . .
RUN CGO_ENABLED=0 go build -o tentask .

FROM alpine:latest as prod
WORKDIR /data/
COPY --from=builder /data/tentask /data/eztask.yaml ./
ENTRYPOINT ["/data/tentask", "start", "-c", "/data/eztask.yaml"]