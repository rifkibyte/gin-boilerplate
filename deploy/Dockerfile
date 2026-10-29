FROM golang:1.26-alpine AS build

WORKDIR /app

# Cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Build the binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/seed ./cmd/seed

# --- runtime image ---
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/api /app/api
COPY --from=build /out/seed /app/seed
COPY --from=build /app/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/api"]
