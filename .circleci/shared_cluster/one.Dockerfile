# syntax=docker/dockerfile:1

FROM ubuntu:22.04
WORKDIR /app

COPY wait-and-clean /wait-and-clean

EXPOSE 8080
