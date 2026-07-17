# 🚀 Go Minimal Backend (Clean Architecture)

โปรเจกต์นี้คือ Template สำหรับสร้าง Backend ด้วย **Go 1.26.1** แบบ Minimal ไม่พึ่งพา Framework หนักๆ (ใช้ Standard Library `net/http`) แต่จัดโครงสร้างแบบ **Clean Architecture** รัดกุมเหมือน .NET เพื่อให้โปรเจกต์ขยายได้ง่ายและดูแลรักษาได้ในระยะยาว

## 📂 โครงสร้างโฟลเดอร์โดยสังเขป
โปรเจกต์นี้แบ่งออกเป็น 3 โฟลเดอร์หลัก:
1. `cmd/` - จุดเริ่มต้นของโปรแกรม (ไฟล์ `main.go`)
2. `internal/` - โค้ดส่วนตัวของแอปเรา (คนอื่น import ไปใช้ไม่ได้) **หัวใจหลักอยู่ที่นี่**
3. `pkg/` - โค้ดเครื่องมือส่วนกลาง (เช่นตัวแปลง JSON, การทำ JWT)

## 🔄 การไหลของข้อมูล (Data Flow)
เวลาที่มี Request เข้ามาผ่าน API ข้อมูลจะไหลเป็นชั้นๆ ดังนี้ (แบบเดินรถทางเดียว):

**HTTP Request** -> `Delivery` (รับ/ส่งข้อมูล) -> `Usecase` (คิดคำนวณ) -> `Repository` (คุยกับ DB)

> 💡 **กฎเหล็ก**: วงแหวนวงนอกสุดรู้จักรวงใน แต่วงในสุดห้ามรู้จักวงนอก (เช่น `Domain` ห้าม import `Usecase`, และ `Usecase` ห้าม import `Delivery`)

## 🔐 ระบบสิทธิ์ผู้ใช้งาน (RBAC)
ระบบได้ถูกติดตั้ง **Role-Based Access Control** เรียบร้อยแล้ว:
- **Admin**: ถูกสร้างไว้ให้อัตโนมัติ (Seed Data) ตอนรันระบบครั้งแรกด้วย Database Migration คุณสามารถ Login ด้วย `username`: `admin` และ `password`: `admin123` เพื่อรับสิทธิ์ Admin ทันที (ไม่ต้องสมัคร)
- **User**: ทุกๆ ครั้งที่มีการสมัครสมาชิก (Register) ทุกคนจะได้สิทธิ์เป็น `user` ธรรมดาหมด 
- ทุกๆ ครั้งที่ Login ตัว JWT Token จะฝังค่า Role เอาไว้ให้ 
- **ตัวอย่างการห้ามบาง Route:** เส้นทาง `DELETE /api/notes/{id}` ถูกจำกัดสิทธิ์ใน `router.go` ให้เข้าถึงได้เฉพาะคนที่ถือ Token ที่มีสิทธิ์ `admin` เท่านั้น (ผ่าน `middleware.RequireRole("admin")`)


## 🛠 คำสั่ง Go ที่ใช้บ่อยในโปรเจกต์นี้
รวบรวมคำสั่งพื้นฐานและคำสั่งที่ต้องใช้เป็นประจำ:

```bash
# 1. รันโปรเจกต์ (สำหรับตอนพัฒนา)
go run cmd/api/main.go

# 2. คอมไพล์โปรเจกต์ (เอาไว้ไปรันจริงบน Production)
go build -o api.exe cmd/api/main.go 
# บน Linux/Mac จะใช้: go build -o api cmd/api/main.go

# 3. โหลด Package ทั้งหมดให้ครบถ้วน (เผื่อไปเปิดเครื่องอื่น)
go mod tidy

# 4. ทดสอบความถูกต้องของ Syntax และโค้ดเบื้องต้น
go vet ./...

# 5. รัน Unit Test (ถ้ามีการเขียนไฟล์ _test.go ในอนาคต)
go test ./... -v
```

## 🚀 วิธีเริ่มต้น (Quick Start)
1. สั่งรัน Database ขึ้นมาก่อนด้วย `docker compose up -d`
2. ตรวจสอบตั้งค่าฐานข้อมูลในไฟล์ `.env` (สลับ username/password ให้ตรงกับเครื่องคุณ ถ้าไม่ได้ใช้ค่า default ของ Docker)
3. รันคำสั่ง `go run cmd/api/main.go`
4. ลองยิง API ผ่าน Postman ได้เลย!

*(เข้าไปอ่าน `README.md` ย่อยตามแต่ละโฟลเดอร์เพื่อทำความเข้าใจอินเทอร์นัลแต่ละส่วนได้เลย!)*
