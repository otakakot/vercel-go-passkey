FROM golang:1.23.0-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

ARG workdir

WORKDIR /app/${workdir}

CMD ["air"]
