FROM docker.io/golang:1.22.3-alpine3.20 AS builder

RUN apk --no-cache add make git
WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build and install
COPY . .
RUN CGO_ENABLED=0 make build

FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/gemportal /bin/gemportal

WORKDIR /bin
USER nobody
ENTRYPOINT ["/bin/gemportal"]
