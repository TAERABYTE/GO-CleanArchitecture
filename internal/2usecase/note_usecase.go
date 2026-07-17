package usecase

import (
	"context"

	"go-minimal-backend/internal/4domain"
)

// noteUsecase คือ struct ที่ implement domain.NoteUsecase (usecase layer)
// เก็บ noteRepo ไว้ข้างใน แต่ประกาศ type เป็น domain.NoteRepository ซึ่งเป็น "interface"
// ไม่ใช่ struct ของ Postgres โดยตรง (*postgres.noteRepository)
// เพราะฉะนั้น usecase layer นี้จะไม่รู้เลยว่าข้อมูลจริงๆ มาจาก Postgres, MySQL หรือ mock
type noteUsecase struct {
	noteRepo domain.NoteRepository
}

// NewNoteUsecase คือ constructor ของ noteUsecase
// รับ nr (ของจริงคือ *postgres.noteRepository ที่สร้างจาก NewNoteRepository ใน note_repo.go)
// เข้ามาทาง parameter type domain.NoteRepository (dependency injection)
// แล้วห่อใส่ noteUsecase คืนกลับไปเป็น interface domain.NoteUsecase
//
// จุดนี้คือหัวใจของ dependency injection: ตอนเริ่มโปรแกรม (main.go) จะเป็นคนเลือกว่า
// จะส่ง repository ตัวไหนเข้ามา (Postgres จริง หรือ mock ตอนเทส) usecase ไม่ต้องแก้โค้ดเลย
func NewNoteUsecase(nr domain.NoteRepository) domain.NoteUsecase {
	return &noteUsecase{
		noteRepo: nr,
	}
}

// Create รับคำสั่งสร้างโน้ตจาก handler ชั้นบน แล้ว "ส่งต่อ" ไปให้ noteRepo.Create ทำงานจริง
// (noteRepo.Create คือเมธอดที่ implement อยู่ใน internal/3repository/postgres/note_repo.go)
// ตอนนี้ยังไม่มี business logic เพิ่มเติม (เช่น validate) จึงแค่ forward ไปตรงๆ
func (u *noteUsecase) Create(ctx context.Context, note *domain.Note) error {
	return u.noteRepo.Create(ctx, note)
}

// GetByID ส่งต่อไปเรียก noteRepo.GetByID (ใน note_repo.go) เพื่อดึงโน้ต 1 รายการ
// โดยเช็ค id คู่กับ userID เพื่อกันไม่ให้ดึงโน้ตของ user คนอื่นได้ (ตรรกะสิทธิ์นี้จริงๆ
// ถูกบังคับด้วย SQL WHERE ที่ชั้น repository อีกที)
func (u *noteUsecase) GetByID(ctx context.Context, id int, userID int) (*domain.Note, error) {
	return u.noteRepo.GetByID(ctx, id, userID)
}

// List เรียก noteRepo.ListByUserID (ใน note_repo.go) เพื่อดึงโน้ตทั้งหมดของ user คนเดียว
// สังเกตว่าชื่อเมธอดฝั่ง usecase (List) กับฝั่ง repository (ListByUserID) ไม่จำเป็นต้องชื่อเหมือนกัน
// เพราะคนละ interface กัน (domain.NoteUsecase vs domain.NoteRepository)
func (u *noteUsecase) List(ctx context.Context, userID int) ([]*domain.Note, error) {
	return u.noteRepo.ListByUserID(ctx, userID)
}

// Update ส่งต่อไปเรียก noteRepo.Update (ใน note_repo.go)
// คอมเมนต์เดิมบอกว่า "Let repo handle finding by id and userid" หมายถึงว่า usecase ชั้นนี้
// ไม่ต้องเช็คเองว่าโน้ตนี้มีอยู่จริงไหมหรือเป็นของ user คนนี้หรือเปล่า ปล่อยให้ SQL
// WHERE id = ... AND user_id = ... ในชั้น repository เป็นคนเช็คแทน (ลด logic ซ้ำซ้อน)
func (u *noteUsecase) Update(ctx context.Context, input *domain.Note) error {
	// Let repo handle finding by id and userid
	return u.noteRepo.Update(ctx, input)
}

// Delete ส่งต่อไปเรียก noteRepo.Delete (ใน note_repo.go) ให้ลบโน้ตตาม id + userID
func (u *noteUsecase) Delete(ctx context.Context, id int, userID int) error {
	return u.noteRepo.Delete(ctx, id, userID)
}
