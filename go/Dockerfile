FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go mod download

RUN apk add --no-cache gcc musl-dev

EXPOSE 9000

RUN addgroup -S mercari && adduser -S trainee -G mercari
# RUN chown -R trainee:mercari /path/to/db

USER trainee

CMD ["go", "run", "./cmd/api"]
