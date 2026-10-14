FROM golang:1.25.4 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/auction cmd/auction/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=build /app/auction /app/auction
COPY cmd/auction/.env /app/cmd/auction/.env

EXPOSE 8080

ENTRYPOINT ["/app/auction"]
