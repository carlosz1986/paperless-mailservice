FROM golang:1.22-alpine
WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
 
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /paperless-mailservice

# Run
CMD ["/paperless-mailservice"]
