# build the server binary
FROM golang:1.13.1 AS builder
LABEL stage=server-intermediate
WORKDIR /go/src/github.com/amikhailau/users-service

COPY . .
RUN go build -o bin/server ./cmd/server

# copy the server binary from builder stage; run the server binary
FROM alpine:latest AS runner
WORKDIR /bin

# Go programs require libc
RUN mkdir -p /lib64 && \
    ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY --from=builder /go/src/github.com/amikhailau/users-service/bin/server .
COPY pkg/pb/*.swagger.json www/swagger.json
COPY --from=builder /go/src/github.com/amikhailau/users-service/db/migrations /bin/db/migrations/
COPY --from=builder /go/src/github.com/amikhailau/users-service/pkg/auth auth/
#ENTRYPOINT server --gateway.swaggerFile www/swagger.json --session.key.private.path /bin/auth/private_unencrypted.pem --session.key.public.path /bin/auth/public.pem --server.port $PORT
ENTRYPOINT ["server", "--gateway.swaggerFile", "www/swagger.json", "--session.key.private.path", "auth/private_unencrypted.pem", "--session.key.public.path", "auth/public.pem"]
