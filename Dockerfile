FROM golang:1.23 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server cmd/server/main.go

FROM alpine:edge

WORKDIR /app

COPY --from=build /app/server .

CMD [ "/app/server" ]