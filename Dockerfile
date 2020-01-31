FROM golang:1.12.3
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api-server

FROM goodwithtech/dockle:v0.2.4
WORKDIR /app

FROM aquasec/trivy
WORKDIR /app

COPY --from=0 /app/api-server .
COPY --from=1 /usr/local/bin/dockle /usr/local/bin/dockle

COPY versions.json .

# Build vulnerabilty cache
RUN trivy --download-db-only

EXPOSE 8000

# Remove entrypoint from parent image
ENTRYPOINT []

CMD ["./api-server"]
