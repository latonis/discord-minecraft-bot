FROM golang:1.20

WORKDIR /bot

COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN go build
CMD "./discord-minecraft-bot"