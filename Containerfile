FROM registry.redhat.io/rhel9/go-toolset:latest AS builder
WORKDIR /opt/app-root/src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o app main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:latest
WORKDIR /opt/app-root/

#复制CA证书
COPY --from=builder /etc/pki/tls/certs/ca-bundle.crt /etc/pki/tls/certs/
COPY --from=builder /opt/app-root/src/app .
EXPOSE 8080
CMD ["/opt/app-root/app"]
