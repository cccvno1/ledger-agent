FROM node:22-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm run build

FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/server .
COPY configs/ ./configs/

EXPOSE 8080

ENTRYPOINT ["./server"]
