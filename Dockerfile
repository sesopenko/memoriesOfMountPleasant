FROM golang:1.19-bullseye

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

COPY static static
RUN CGO_ENABLED=0 GOOS=linux go build -o /serve

EXPOSE 8080/tcp
CMD ["/serve"]