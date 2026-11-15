FROM golang:1.27.4-alpine3.24 AS build
WORKDIR /app

ARG GO_TAGS=""

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -tags="${GO_TAGS}" -o /usr/local/bin/app ./cmd/coordinator

FROM alpine:3.24
WORKDIR /app

RUN apk add openssh && ssh-keygen -t ed25519 -f "/root/.ssh/id_ed25519" -N ""

COPY --from=build /usr/local/bin/app .

ENTRYPOINT ["./app"]
