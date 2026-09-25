# Build context: the repository root.
FROM golang:1.27 AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# A static binary needs no libc: distroless keeps the image to the binary,
# CA certificates and a non-root user.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
