FROM golang:1.16.4-alpine3.12 AS builder

RUN apk --no-cache add ca-certificates make git
WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build and install
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/gemportal /bin/gemportal
ENTRYPOINT ["/bin/gemportal"]