FROM golang:latest

WORKDIR /pb

# Copy Go files
COPY main.go go.mod go.sum ./

# use mattn/go-sqlite3 driver
ENV CGO_ENABLED=1

# Build the program
RUN go build -o dietly-pb

EXPOSE 8080

CMD ["./dietly-pb", "serve", "--http=0.0.0.0:8080", "--encryptionEnv=PB_ENCRYPTION_KEY"]
