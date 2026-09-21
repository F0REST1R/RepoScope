# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS api-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/reposcope-api ./cmd/api

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=api-build /out/reposcope-api /usr/local/bin/reposcope-api
USER app
EXPOSE 8080
ENTRYPOINT ["reposcope-api"]

FROM node:24-alpine AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM nginx:1.27-alpine AS web
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=web-build /src/web/dist /usr/share/nginx/html
COPY docs/openapi.yaml /usr/share/nginx/html/docs/openapi.yaml
EXPOSE 80
