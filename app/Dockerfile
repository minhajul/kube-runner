# syntax=docker/dockerfile:1.7

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./

# ---- runtime stage ----
# Scratch is small but has no shell, no time. Distroless static carries
# just enough (ca-certificates, /etc/passwd, tzdata) to make the binary
# debuggable in-cluster without bloating it.
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENV LISTEN_ADDR=:8080
ENTRYPOINT ["/app"]