FROM golang:1.27.4-alpine3.24 AS gobuild
WORKDIR /app

ARG GO_TAGS=""

COPY coordinator/go.mod coordinator/go.sum ./
RUN go mod download

COPY coordinator/ .
RUN go build -v -tags="${GO_TAGS}" -o /usr/local/bin/app ./cmd/coordinator

FROM python:3.12-alpine AS pythonbuild
WORKDIR /app

RUN apk add --no-cache binutils gcc musl-dev libffi-dev

COPY agent/ .
RUN pip install --no-cache-dir -r requirements.txt pyinstaller

RUN pyinstaller --onefile --name=agent --distpath=/usr/local/bin main.py

FROM alpine:3.24
WORKDIR /app

RUN apk add openssh && ssh-keygen -t ed25519 -f "/root/.ssh/id_ed25519" -N ""

COPY --from=pythonbuild /usr/local/bin/agent .

COPY --from=gobuild /usr/local/bin/app .

ENTRYPOINT ["./app"]
