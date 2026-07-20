# ---- Build stage: compile เป็น binary เดียว ไม่พก Go toolchain ติดไปด้วย ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

# copy แค่ go.mod/go.sum ก่อนเพื่อให้ layer นี้ cache ไว้ได้ (ไม่ต้อง go mod download ใหม่
# ทุกครั้งที่แก้โค้ด ตราบใดที่ dependency ไม่เปลี่ยน)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 ให้ได้ static binary (pgx เป็น pure Go driver อยู่แล้ว ไม่ต้องพึ่ง CGO)
# รันบน alpine (musl) ได้โดยไม่ติด glibc
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# ---- Run stage: image เล็กสุด มีแค่ binary + ca-certificates ----
FROM alpine:3.20

# ต้องมี ca-certificates ไว้ verify TLS เผื่อ DB หรือ service อื่นต่อผ่าน sslmode=require ในอนาคต
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/api ./api

EXPOSE 5000

ENTRYPOINT ["./api"]
