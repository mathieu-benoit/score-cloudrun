FROM golang:1.27-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS builder

ARG VERSION

# Set the current working directory inside the container.
WORKDIR /go/src/github.com/score-spec/score-cloudrun

# Copy just the module bits
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project and build it.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/score-spec/score-cloudrun/internal/version.Version=${VERSION}" -o /usr/local/bin/score-cloudrun ./cmd/score-cloudrun

# We can use gcr.io/distroless/static since we don't rely on any linux libs or state, but we need ca-certificates to connect to https/oci with the init command.
FROM gcr.io/distroless/static:530158861eebdbbf149f7e7e67bfe45eb433a35c@sha256:5c7e2b465ac6a2a4e5f4f7f722ce43b147dabe87cb21ac6c4007ae5178a1fa58

# Set the current working directory inside the container.
WORKDIR /score-cloudrun

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-cloudrun /usr/local/bin/score-cloudrun

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-cloudrun"]