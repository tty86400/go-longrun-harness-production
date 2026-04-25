FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go test ./... && go build -o /out/harness ./cmd/harness

FROM alpine:3.20
RUN adduser -D -u 10001 harness
USER harness
WORKDIR /app
COPY --from=build /out/harness /app/harness
COPY examples /app/examples
COPY docs /app/docs
EXPOSE 8080
ENTRYPOINT ["/app/harness"]
