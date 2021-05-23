FROM golang:1.16

WORKDIR /go/src/app

# Download dependencies
COPY go.mod go.sum ./
RUN go get ./...

# Build and install
COPY . .
RUN go install -v ./...

CMD ["gemportal"]