# --------- build stage ---------
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/shorty ./cmd/server

# --------- runtime stage ---------
FROM alpine:3.20
RUN adduser -D -u 10001 appuser
WORKDIR /home/appuser
COPY --from=build /out/shorty /usr/local/bin/shorty
USER appuser
ENV PORT=8080
EXPOSE 8080
CMD ["/usr/local/bin/shorty"]
