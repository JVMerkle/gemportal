FROM golang:1.16 AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go get ./...

# Build and install
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" .

FROM scratch
COPY --from=builder /app/gemportal /bin/gemportal
ENTRYPOINT ["/bin/gemportal"]