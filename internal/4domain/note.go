package domain

import (
	"context"
	"time"
)

// Note คือโมเดลข้อมูลหลักของฟีเจอร์ "โน้ต" ใช้เป็นตัวกลางส่งผ่านข้อมูลไปมา
// ระหว่างทุก layer (handler <-> usecase <-> repository <-> DB)
// json tag ท้าย field ใช้กำหนดว่าตอน encode/decode เป็น JSON จะใช้ชื่อ key อะไร
// (เช่น struct field CreatedAt จะกลายเป็น "created_at" ใน JSON ที่ client เห็น)
type Note struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`    // เจ้าของโน้ต ใช้เช็คสิทธิ์ว่าใครมีสิทธิ์อ่าน/แก้/ลบได้บ้าง
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"` // ถูกกำหนดค่าตอน INSERT โดย Postgres เอง (ดู Create ใน note_repo.go)
	UpdatedAt time.Time `json:"updated_at"` // ถูกกำหนดค่าใหม่ทุกครั้งที่ UPDATE โดย Postgres เอง
}

// NoteRepository คือ interface (สัญญา) ของชั้นเข้าถึงข้อมูล (data access layer)
// นิยามไว้แค่ "ต้องมีเมธอดอะไรบ้าง" โดยไม่สนใจว่าข้างในจะคุยกับ DB ตัวไหนหรือยังไง
// ตัวที่ implement จริงคือ *noteRepository ใน internal/3repository/postgres/note_repo.go
// การแยกเป็น interface แบบนี้ทำให้ชั้น usecase ไม่ผูกติดกับ Postgres โดยตรง
// (จะสลับไปใช้ DB อื่น หรือ mock ตอนเทส ก็แค่เขียน struct ใหม่ให้มีเมธอดครบตามนี้)
type NoteRepository interface {
	Create(ctx context.Context, note *Note) error
	GetByID(ctx context.Context, id int, userID int) (*Note, error)
	ListByUserID(ctx context.Context, userID int) ([]*Note, error)
	Update(ctx context.Context, note *Note) error
	Delete(ctx context.Context, id int, userID int) error
}

// NoteUsecase คือ interface (สัญญา) ของชั้น business logic
// ตัวที่ implement จริงคือ *noteUsecase ใน internal/2usecase/note_usecase.go
// ชั้น handler (internal/1delivery/http/handler/note_handler.go) จะเรียกใช้ผ่าน interface นี้
// เท่านั้น ไม่รู้จัก noteUsecase struct จริงๆ เลย (ลด coupling ระหว่างชั้น HTTP กับ business logic)
//
// หมายเหตุ: ชื่อเมธอดของ NoteUsecase ไม่จำเป็นต้องตรงกับ NoteRepository เป๊ะ
// (เช่น List ที่นี่ เทียบเท่ากับ ListByUserID ของ NoteRepository) เพราะเป็นคนละ interface กัน
type NoteUsecase interface {
	Create(ctx context.Context, input *Note) error
	GetByID(ctx context.Context, id int, userID int) (*Note, error)
	List(ctx context.Context, userID int) ([]*Note, error)
	Update(ctx context.Context, input *Note) error
	Delete(ctx context.Context, id int, userID int) error
}
