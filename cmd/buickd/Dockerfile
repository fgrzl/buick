# Build from the repository root:
#   docker build -f cmd/buickd/Dockerfile -t buickd .
FROM golang:trixie AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath -o /buickd \
    -ldflags "-s -w -X github.com/fgrzl/buick/internal/buildinfo.Version=${VERSION} -X github.com/fgrzl/buick/internal/buildinfo.Commit=${GIT_COMMIT} -X github.com/fgrzl/buick/internal/buildinfo.BuildDate=${BUILD_DATE}" \
    ./cmd/buickd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /buickd /buickd

USER nonroot:nonroot
EXPOSE 80 443

ENTRYPOINT ["/buickd"]
CMD ["--config", "/etc/buick/buick.yml"]
