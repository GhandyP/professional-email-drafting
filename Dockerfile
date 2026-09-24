# syntax=docker/dockerfile:1

# ---- build stage -----------------------------------------------------------
FROM golang:1.26.5 AS build
WORKDIR /src

# Dependencies first, so this layer stays cached across source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/professional-email-drafting .

# ---- runtime stage ---------------------------------------------------------
# Distroless: no shell, no package manager, non-root user. The CA bundle is copied
# explicitly because drafting calls the Gemini API over HTTPS.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /work
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build --chown=nonroot:nonroot /out/professional-email-drafting /work/professional-email-drafting
COPY --from=build --chown=nonroot:nonroot /src/testdata /work/testdata
USER nonroot:nonroot
ENTRYPOINT ["/work/professional-email-drafting"]
