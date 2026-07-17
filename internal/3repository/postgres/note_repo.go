package postgres

import (
	"context"
	"errors"

	"go-minimal-backend/internal/4domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// noteRepository คือ struct ที่เก็บ "ตัวเชื่อมต่อฐานข้อมูล" ไว้ข้างใน
// เพื่อให้ทุกเมธอดของ struct นี้ (Create, GetByID, ...) เอาไปใช้ยิง query ได้
type noteRepository struct {
	db *pgxpool.Pool // pgxpool.Pool คือ connection pool ของ pgx (ไลบรารีคุย Postgres) แทนที่จะเปิด/ปิด connection ทุกครั้ง มันจะสำรอง connection ไว้ใช้ซ้ำหลายๆ ตัวพร้อมกัน
}

// NewNoteRepository คือ constructor (ฟังก์ชันสร้าง object)
// รับ pool ที่เชื่อมต่อ DB ไว้แล้วเข้ามา แล้วห่อใส่ noteRepository คืนออกไป
//
// จุดสำคัญ: return type ประกาศเป็น domain.NoteRepository (interface จาก domain layer)
// ไม่ใช่ *noteRepository (struct จริง) เพื่อบังคับให้โค้ดฝั่งที่เรียกใช้ (usecase)
// มองเห็นแค่ "สัญญา" ของ interface เท่านั้น ไม่รู้จักรายละเอียดว่าข้างในเป็น Postgres
func NewNoteRepository(db *pgxpool.Pool) domain.NoteRepository {
	return &noteRepository{db: db}
}

// Create = เพิ่มโน้ตใหม่ 1 รายการลงตาราง notes
// รับ note ที่ผู้ใช้กรอกมา (มี UserID, Title, Content) แล้วบันทึกลง DB
// เมื่อบันทึกสำเร็จ DB จะ generate ID, created_at, updated_at ให้อัตโนมัติ
// โค้ดจึงต้อง "อ่านค่ากลับ" มาใส่ใน note ที่ส่งเข้ามา (pointer) ด้วย
func (r *noteRepository) Create(ctx context.Context, note *domain.Note) error {
	// RETURNING id, created_at, updated_at คือให้ Postgres ส่งค่า column
	// ที่มันสร้างขึ้นเองกลับมาในคำสั่ง INSERT เดียวกัน ไม่ต้อง SELECT ซ้ำอีกรอบ
	query := `INSERT INTO notes (user_id, title, content) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`

	// QueryRow ใช้ยิงคำสั่งที่คาดว่าจะได้ผลลัพธ์กลับมา "แถวเดียว"
	// $1, $2, $3 จะถูกแทนด้วย note.UserID, note.Title, note.Content ตามลำดับ (ป้องกัน SQL injection)
	// .Scan(...) คือ "แกะ" ค่าจากแถวผลลัพธ์ ใส่กลับเข้า field ของ note ผ่าน pointer (&note.ID ฯลฯ)
	err := r.db.QueryRow(ctx, query, note.UserID, note.Title, note.Content).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)

	// ถ้า Scan ผิดพลาด (เช่น constraint ผิด, connection หลุด) จะได้ error กลับมา
	// ถ้าไม่ผิดพลาด err จะเป็น nil และ note ที่ส่งเข้ามาจะถูกเติม ID/CreatedAt/UpdatedAt ครบแล้ว
	return err
}

// GetByID = ค้นหาโน้ต 1 รายการ ด้วย id และ userID (กันไม่ให้ user คนอื่นดึงโน้ตของคนอื่นได้)
func (r *noteRepository) GetByID(ctx context.Context, id int, userID int) (*domain.Note, error) {
	query := `SELECT id, user_id, title, content, created_at, updated_at FROM notes WHERE id = $1 AND user_id = $2`

	// QueryRow คืนค่ามาแถวเดียว (หรือไม่มีเลยถ้าไม่เจอ)
	row := r.db.QueryRow(ctx, query, id, userID)

	// สร้างตัวแปร note เปล่าไว้รอรับค่า
	var note domain.Note
	// Scan แกะค่าจากแถวใส่ทีละ field ตามลำดับที่ SELECT มา (ต้องเรียงให้ตรงกัน)
	err := row.Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		// pgx.ErrNoRows คือ error เฉพาะของ pgx ที่บอกว่า "query ไม่ error แต่ไม่เจอแถวไหนเลย"
		// errors.Is ใช้เทียบว่า err ตัวนี้ "คือ" หรือ "ห่อ" ErrNoRows อยู่หรือเปล่า
		if errors.Is(err, pgx.ErrNoRows) {
			// แปลง error เฉพาะของ pgx ให้เป็น error กลางของ domain layer (domain.ErrNotFound)
			// เพื่อไม่ให้ชั้น usecase/handler ต้องรู้จัก error ของ pgx โดยตรง (ลด coupling)
			return nil, domain.ErrNotFound
		}
		// error อื่นๆ ที่ไม่ใช่ "ไม่เจอแถว" (เช่น DB ล่ม) ส่งกลับตรงๆ
		return nil, err
	}

	// เจอข้อมูลและ scan สำเร็จ คืน pointer ของ note กลับไป
	return &note, nil
}

// ListByUserID = ดึงโน้ตทั้งหมดของ user คนหนึ่ง เรียงจากใหม่ไปเก่า
func (r *noteRepository) ListByUserID(ctx context.Context, userID int) ([]*domain.Note, error) {
	// ORDER BY created_at DESC = เรียงตามวันที่สร้าง จากล่าสุดไปเก่าสุด
	query := `SELECT id, user_id, title, content, created_at, updated_at FROM notes WHERE user_id = $1 ORDER BY created_at DESC`

	// Query (ต่างจาก QueryRow) ใช้ตอนคาดว่าจะได้ผลลัพธ์กลับมา "หลายแถว"
	// rows คือ cursor/ตัวชี้ตำแหน่ง ไว้ไล่อ่านทีละแถว ไม่ได้โหลดข้อมูลทั้งหมดเข้า memory ทันที
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	// defer = สั่งให้รันตอนฟังก์ชันจบการทำงาน (ไม่ว่าจะ return ตรงไหนก็ตาม)
	// ปิด rows เพื่อคืน connection กลับเข้า pool ป้องกัน resource leak
	defer rows.Close()

	// เตรียม slice เปล่าไว้เก็บผลลัพธ์ทั้งหมด
	var notes []*domain.Note

	// rows.Next() เลื่อน cursor ไปแถวถัดไป คืน true ถ้ายังมีข้อมูลเหลือ, false ถ้าหมดแล้ว
	for rows.Next() {
		// ประกาศ note ใหม่ "ในลูปทุกรอบ" สำคัญมาก เพราะถ้าประกาศไว้นอกลูป
		// ทุก pointer ที่ append เข้า slice จะชี้ไปที่ตัวแปรเดียวกัน ทำให้ข้อมูลผิดเพี้ยนหมด
		var note domain.Note
		if err := rows.Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		// เก็บ pointer ของ note รอบนี้เข้า slice
		notes = append(notes, &note)
	}

	// rows.Next() คืน false ได้ทั้งกรณี "อ่านจบปกติ" และ "error ระหว่างดึงข้อมูล"
	// ต้องเช็ค rows.Err() เพิ่มหลังลูปจบเสมอ เพื่อจับ error ที่อาจซ่อนอยู่ระหว่างวนลูป
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// คืนรายการโน้ตทั้งหมด (ถ้า user ไม่มีโน้ตเลย notes จะเป็น nil ไม่ถือว่า error)
	return notes, nil
}

// Update = แก้ไข title/content ของโน้ตที่มีอยู่ ด้วยเงื่อนไข id + userID (กันแก้โน้ตคนอื่น)
func (r *noteRepository) Update(ctx context.Context, note *domain.Note) error {
	// CURRENT_TIMESTAMP ให้ Postgres เป็นคนตั้งเวลาปัจจุบันเองตอน UPDATE (แม่นยำกว่าส่งเวลาจากฝั่งแอปมา)
	// RETURNING updated_at ให้ส่งเวลาที่เพิ่งอัปเดตกลับมา เพื่อเอาไปเก็บใน note.UpdatedAt ต่อ
	query := `UPDATE notes SET title = $1, content = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3 AND user_id = $4 RETURNING updated_at`

	err := r.db.QueryRow(ctx, query, note.Title, note.Content, note.ID, note.UserID).Scan(&note.UpdatedAt)
	if err != nil {
		// ถ้า RETURNING ไม่มีแถวให้ scan แปลว่า WHERE id = ... AND user_id = ... ไม่ match แถวไหนเลย
		// (โน้ตไม่มีอยู่จริง หรือไม่ใช่เจ้าของ) ให้แปลงเป็น domain.ErrNotFound เหมือนใน GetByID
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

// Delete = ลบโน้ต ด้วยเงื่อนไข id + userID (กันลบโน้ตคนอื่น)
func (r *noteRepository) Delete(ctx context.Context, id int, userID int) error {
	query := `DELETE FROM notes WHERE id = $1 AND user_id = $2`

	// Exec ใช้กับคำสั่งที่ไม่ต้องการผลลัพธ์เป็นแถวข้อมูลกลับมา (INSERT/UPDATE/DELETE ที่ไม่มี RETURNING)
	// tag คือ "ป้ายกำกับผลลัพธ์" ที่บอกว่าคำสั่งนี้ส่งผลกระทบกี่แถว
	tag, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	// ถ้าไม่มีแถวไหนถูกลบเลย แปลว่าไม่เจอโน้ตตาม id/userID นี้ (หรือไม่ใช่เจ้าของ)
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
