FROM golang:1.24-alpine

RUN apk add --no-cache gcc musl-dev sqlite

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN addgroup -S mercari && adduser -S trainee -G mercari
RUN chown -R trainee:mercari ./images ./db

USER trainee

CMD ["go", "run", "./cmd/api/main.go"]
