FROM golang:1.23.4-alpine3.21

WORKDIR /app

COPY . .

RUN go build -o server ./cmd/main.go

EXPOSE 8080

CMD ["./server"]