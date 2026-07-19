FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/server ./server
ENV PORT=8080
ENV DB_PATH=/data/sensors.db
EXPOSE 8080
ENTRYPOINT ["./server"]
