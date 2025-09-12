# Build stage
FROM golang:1 as build

WORKDIR /go/src/app
COPY main.go .
COPY go.mod .
COPY go.sum .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

# Final stage
FROM gcr.io/distroless/static-debian13
COPY --from=build /go/bin/app /
ENTRYPOINT ["/app"]
