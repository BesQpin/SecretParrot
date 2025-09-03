# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w" -o /out/secret-parrot ./cmd/secret-parrot

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/secret-parrot /secret-parrot
USER nonroot:nonroot
ENTRYPOINT ["/secret-parrot"]