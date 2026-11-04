FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server && CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && CGO_ENABLED=0 go build -trimpath -o /out/admin ./cmd/admin

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S taskflow && adduser -S -G taskflow taskflow
COPY --from=build /out/server /out/migrate /out/admin /app/
USER taskflow
EXPOSE 8081
CMD ["/app/server"]
