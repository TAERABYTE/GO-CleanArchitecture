# 🚀 Go Minimal Backend (Clean Architecture)

## 🐳 รันทั้ง API + Database ผ่าน Docker

ตอนนี้ `docker-compose.yml` มี 2 service: **`db`** (PostgreSQL) กับ **`api`** (ตัว Go เอง) รันพร้อมกันได้เลยทั้งคู่ ไม่ต้องเปิด terminal แยกมา `go run` เองอีกต่อไป (แต่ยังใช้ `go run` ตอน dev ได้ตามปกติถ้าอยากแก้โค้ดแล้วเห็นผลไว)

```bash
# Build image ของ api ใหม่ + สั่งรันทั้งคู่ (ใช้ตอนแรก หรือทุกครั้งที่แก้โค้ด Go)
docker compose up -d --build

# เช็คสถานะ container ทั้งหมด
docker compose ps

# ดู log ของ api แบบ real-time (ใช้แทน terminal ที่เคย `go run` เอง)
docker compose logs -f api

# ปิดทั้งคู่ (ข้อมูลใน DB ยังอยู่ เพราะเก็บใน volume แยก)
docker compose down
```

---

## 📖 อธิบายทีละขั้น ว่าแต่ละไฟล์ทำอะไร

### 1. `Dockerfile` — สูตรสร้าง image ของ Go API

Dockerfile แบ่งเป็น **2 stage** เหมือนทำอาหาร: stage แรกคือ "ครัวที่มีเครื่องมือครบ" ใช้ทำอาหารเสร็จแล้วก็ทิ้งครัวไป เอาแค่ "จานอาหารที่เสร็จแล้ว" ไปเสิร์ฟจริง

```dockerfile
FROM golang:1.26-alpine AS builder   # (1) stage "ครัว" มี Go compiler ครบ
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download                  # (2) โหลด dependency ก่อน (cache ไว้ ไม่ต้องโหลดซ้ำทุกครั้ง)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api   # (3) compile ได้ binary ไฟล์เดียว

FROM alpine:3.20                     # (4) stage "จานเสิร์ฟ" เอาแค่ binary ไปวาง ไม่มี Go compiler ติดไปด้วย
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/api ./api   # (5) copy เฉพาะ binary จาก stage แรกมา
ENTRYPOINT ["./api"]
```

**ทำไมต้องแยก 2 stage:** ถ้ารวมเป็น stage เดียว image สุดท้ายจะพก Go compiler (~300MB+) ติดไปด้วยทั้งที่ตอนรันจริงไม่ได้ใช้ compiler แล้ว แยกแบบนี้ image สุดท้ายเหลือแค่ตัว binary + ca-certificates เล็กกว่ามาก โหลด/deploy เร็วกว่า

### 2. `docker-compose.yml` — สั่งให้ 2 container คุยกันเป็น

```yaml
services:
  db:
    image: postgres:16-alpine
    ...

  api:
    build:
      context: .
      dockerfile: Dockerfile
    env_file:
      - .env                # (1) ดึงค่า PORT/JWT_SECRET/CORS_ALLOWED_ORIGINS จาก .env มาให้ container
    environment:
      DB_DSN: postgres://postgres:password@db:5432/go_minimal_db?sslmode=disable   # (2)
    depends_on:
      db:
        condition: service_healthy    # (3) รอ db "พร้อมจริงๆ" (ผ่าน healthcheck) ก่อนค่อย start api
```

**(1) `env_file: .env`** — เอาไฟล์ `.env` ที่มีอยู่แล้ว (ที่ใช้ตอน `go run` ปกติ) มาป้อนให้ container ใช้เลย ไม่ต้องเขียนค่าซ้ำสองที่

**(2) ทำไม `DB_DSN` ใน container ต้องเป็น `db` ไม่ใช่ `localhost`** — นี่คือจุดที่มือใหม่งงบ่อยที่สุด: ตอนรันบนเครื่องเราตรงๆ (`go run`) `localhost` หมายถึง "เครื่องเรา" ซึ่งมี Postgres รันอยู่ผ่าน docker แล้วเปิด port 5432 ออกมาให้เรียกได้ แต่พอ Go เองก็ถูกเอาไป**รันอยู่ใน container อีกใบหนึ่งแยกต่างหาก** คำว่า `localhost` จากมุมมองของ container นั้นจะหมายถึง "ตัว container api เอง" ไม่ใช่ container db เลย (container แต่ละใบแยกเครือข่ายกันโดยธรรมชาติ)

Docker Compose แก้ปัญหานี้ด้วยการสร้าง "เครือข่ายภายใน" ให้ทุก service ที่อยู่ใน compose ไฟล์เดียวกัน คุยกันได้โดยเรียกผ่าน **ชื่อ service** ตรงๆ เหมือนเป็นชื่อโดเมนย่อยๆ ในบ้านตัวเอง เพราะฉะนั้น `db` ในที่นี้ = ชื่อ service `db` ที่ประกาศไว้ข้างบน ไม่ใช่ `localhost`

**(3) `depends_on.condition: service_healthy`** — ถ้าไม่มีเงื่อนไขนี้ Docker จะ start `api` พร้อมๆ กับ `db` ทันที แต่ Postgres ต้องใช้เวลา 1-2 วินาที "เริ่มระบบข้างในตัวเอง" ก่อนพร้อมรับ connection จริง ถ้า api ไปเชื่อมต่อเร็วเกินไปจะ connect ไม่ติดแล้ว crash ทันที ต้องรอ `db` "healthy" (ผ่าน `healthcheck: pg_isready` ที่ตั้งไว้ในไฟล์เดียวกัน) ก่อนค่อยปล่อยให้ `api` เริ่มทำงาน

### 3. ทำไมไม่มี `container_name` ให้เห็น

ตอนแรกเคยลองตั้งชื่อ container ตรงๆ (เช่น `go-minimal-db`) แต่พอทดสอบจริงกลับชนกับ container ชื่อเดียวกันจากโปรเจกต์อื่นบนเครื่อง (เผลอ clone template นี้ไว้คนละโฟลเดอร์) เพราะชื่อ container ต้องไม่ซ้ำกันทั้งเครื่อง ไม่ใช่แค่ในโปรเจกต์เดียว

ตอนนี้เลย**ไม่ระบุชื่อเอง** ปล่อยให้ Docker Compose ตั้งชื่อให้อัตโนมัติตามชื่อโฟลเดอร์ของโปรเจกต์ (เช่น `gopher-db-1`, `gopher-api-1`) ถ้า clone repo นี้ไปไว้คนละโฟลเดอร์กี่รอบก็ไม่มีวันชื่อชนกันอีก

---

## 🔧 คำสั่งที่ใช้บ่อยตอนพัฒนา

```bash
docker compose up -d --build     # build ใหม่ + รันทั้งคู่ (ใช้ทุกครั้งที่แก้โค้ด Go แล้วอยากทดสอบผ่าน docker)
docker compose logs -f api       # ดู log ของ api แบบ real-time (Ctrl+C เพื่อออกจากการดู ไม่ได้ปิด container)
docker compose restart api       # รีสตาร์ทแค่ api เฉยๆ (เช่นหลังแก้ .env)
docker compose down              # ปิดทั้งคู่ (ข้อมูลใน DB ยังอยู่ เพราะเก็บใน volume แยกจาก container)
docker compose down -v           # ปิดทั้งคู่ "และลบข้อมูล DB ทิ้งด้วย" (ใช้ตอนอยากเริ่มนับหนึ่งใหม่เท่านั้น)
```

ค่าที่ตั้งไว้ใน `docker-compose.yml` (user/password/database/port) ตรงกับค่า default ใน `.env.example` อยู่แล้ว ใช้คู่กันได้ทันทีโดยไม่ต้องแก้อะไร
