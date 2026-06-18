FROM golang:1.21-alpine

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache git

COPY go.mod ./

RUN go mod download || true

COPY . .

RUN go mod tidy && go build -o main .

EXPOSE 8080

CMD ["/app/main"]
