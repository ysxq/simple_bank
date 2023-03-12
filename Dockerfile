# Build stage
FROM golang:1.20.2-alpine3.17 AS builder
WORKDIR /app
COPY . .
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn
RUN go build -o main
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.15.2/migrate.linux-amd64.tar.gz | tar xvz

# Run stage
FROM alpine:3.17
# 时区
RUN apk --no-cache add tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY app.env .
COPY start.sh .
COPY db/migration ./migration

EXPOSE 8080
CMD ["/app/main"]
ENTRYPOINT [ "/app/start.sh" ]
