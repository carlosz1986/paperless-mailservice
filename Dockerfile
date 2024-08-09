FROM golang:1.19-alpine
WORKDIR /app

COPY go.mod ./
RUN go mod download
 
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /paperless-mailservice

# Run
CMD ["/paperless-mailservice"]
