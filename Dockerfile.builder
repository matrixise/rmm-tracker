# syntax=docker/dockerfile:1.21
#
# Builder base image — pre-downloads all Go modules so the main Dockerfile
# doesn't need to run `go mod download` on every CI build.
#
# Rebuild this image whenever go.mod or go.sum change:
#   task docker:builder:push
#
# Image: matrixise/rmm-tracker-builder:go1.26

FROM --platform=linux/amd64 golang:1.26-alpine

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify
