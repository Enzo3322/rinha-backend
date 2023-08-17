FROM golang:1.16-alpine

RUN apk update && apk add --no-cache git

WORKDIR /

COPY . .

RUN go mod download && go build -o main

CMD ["./main"]
