# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

ARG SERVICE=frontend
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -buildvcs=false -trimpath -o /out/app ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/app /app/app

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/app"]
