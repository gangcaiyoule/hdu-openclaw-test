FROM golang:1.22 AS builder

WORKDIR /build
COPY . /build
RUN go build -o /app/hdu-openclaw ./cmd/server

FROM alpine:3.20

RUN echo 'https://mirrors.ustc.edu.cn/alpine/v3.20/main' > /etc/apk/repositories \
    && echo 'https://mirrors.ustc.edu.cn/alpine/v3.20/community' >> /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app
COPY --from=builder /app/hdu-openclaw /app/hdu-openclaw

EXPOSE 8080
CMD ["./hdu-openclaw"]

