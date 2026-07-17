FROM golang:1.26-alpine AS builder
RUN apk add --no-cache upx ca-certificates
WORKDIR /build
COPY hulk.go go.mod /build/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o hulk . && upx --best -o hulk.packed hulk && mv hulk.packed hulk

FROM scratch
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /build/hulk /hulk
ENTRYPOINT ["/hulk"]
CMD ["--help"]
