# 🚀 Go Minimal Backend (Clean Architecture)

## 🐳 Database ผ่าน Docker
โปรเจกต์นี้มี `docker-compose.yml` ไว้ให้แล้ว แต่ตั้งใจให้รันแค่ **PostgreSQL** เท่านั้น (ตัว Go ยังรันตรงจากเครื่องเหมือนเดิม ยังไม่ได้ dockerize)

```bash
# สั่งรัน Database ขึ้นมา (ครั้งแรกจะดึง image postgres:16-alpine มาก่อน)
docker compose up -d

# เช็คสถานะ container
docker compose ps

# ปิด Database
docker compose down
```

ค่าที่ตั้งไว้ใน `docker-compose.yml` (user/password/database/port) ตรงกับค่า default ใน `.env.example` (`DB_DSN`) อยู่แล้ว ใช้คู่กันได้ทันทีโดยไม่ต้องแก้อะไร

