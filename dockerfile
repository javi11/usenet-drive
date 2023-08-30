FROM golang:1.21-bullseye AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
# Build the Go application inside the container
RUN make build

FROM ghcr.io/linuxserver/baseimage-ubuntu:jammy
ARG DEBIAN_FRONTEND="noninteractive"
ARG BUILD_DATE
ARG VERSION

LABEL build_version="Linuxserver.io version:- ${VERSION} Build-date:- ${BUILD_DATE}"
LABEL maintainer="laris11"

COPY --from=builder /app/bin/ /
# Expose the port that the application will listen on
EXPOSE 8080 8081
VOLUME ["/config", "/nzbs"]

