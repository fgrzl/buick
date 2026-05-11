# Build from the repository root:
#   docker build -f cmd/buick/Dockerfile -t buick .
FROM golang:trixie AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /buick ./cmd/buick

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /buick /buick

USER nonroot:nonroot
EXPOSE 80 443

ENTRYPOINT ["/buick"]
CMD ["--config", "/etc/buick/buick.yml"]
