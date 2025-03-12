# ビルドステージ（CGOの無効化、静的ビルド周り）
FROM golang:1.23.4-alpine AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/main.go

# 実行ステージ
FROM scratch
COPY --from=builder /build/server /app/server

# 非root実行
COPY --from=alpine:latest /etc/passwd /etc/passwd
USER nobody

EXPOSE 8080
WORKDIR /app
CMD ["./server"]
