FROM golang:1.23-alpine AS build
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates
ENV GOPROXY=https://goproxy.cn,direct
ENV GOTOOLCHAIN=auto
COPY go.mod ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /out/lumen ./cmd/lumen

FROM alpine:3.20
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 lumen
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=build /out/lumen /app/lumen
USER lumen
EXPOSE 8080 8081
ENTRYPOINT ["/app/lumen"]
