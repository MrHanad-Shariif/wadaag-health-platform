# Dev-focused image: hot-reloads via air, matches local `go run` behavior.
# A separate multi-stage production build can be added once hosting is decided.
FROM golang:1.26-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
