FROM golang:1.26.2-alpine
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o /app/exe main.go
CMD ["/app/exe"]