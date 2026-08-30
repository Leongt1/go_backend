# ---- build stage -------------------------------------------------------------
# Fat toolchain image: has the Go compiler and module cache. Nothing from this
# stage ships in the final image except the compiled binary we copy out below.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy only the dependency manifests first and download modules. This layer is
# cached and only re-runs when go.mod/go.sum change - editing source code does
# NOT re-download every dependency.
COPY go.mod go.sum ./
RUN go mod download

# Now bring in the source and build. CGO_ENABLED=0 makes a fully static binary
# (no libc dependency), so it runs on a minimal base image.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# ---- run stage ---------------------------------------------------------------
# Tiny base: just enough to run the static binary. ca-certificates is required
# for outbound TLS (Neon Postgres, OpenAI, Resend) - without it those HTTPS
# calls fail with x509 errors.
FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /app/server .

# Run as an unprivileged user, not root.
USER appuser

EXPOSE 8080
CMD ["./server"]
