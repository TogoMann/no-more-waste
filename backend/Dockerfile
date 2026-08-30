FROM golang:1.22-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server .

FROM alpine:3.19
RUN adduser -D -u 10001 appuser && mkdir -p /app/data && chown appuser /app/data
WORKDIR /app
COPY --from=build /out/server /app/server
COPY database/ /app/database/
ENV DB_PATH=/app/data/nomorewaste.db \
    SCHEMA_PATH=/app/database/schema.sql \
    SEED_PATH=/app/database/seed.sql \
    PORT=8080
USER appuser
EXPOSE 8080
CMD ["/app/server"]
