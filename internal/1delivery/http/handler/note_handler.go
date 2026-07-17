package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go-minimal-backend/internal/1delivery/http/middleware"
	"go-minimal-backend/internal/4domain"
	"go-minimal-backend/pkg/response"
)

// จำกัดความยาวให้ตรงกับ constraint ของ DB (notes.title เป็น VARCHAR(255))
// ส่วน content เป็น TEXT ไม่มี limit ระดับ DB แต่กันไว้ระดับ API เพื่อไม่ให้ client ส่ง payload ใหญ่เกินจำเป็น
const (
	noteTitleMaxLen   = 255
	noteContentMaxLen = 10_000
)

// NoteHandler คือชั้นบนสุดของสถาปัตยกรรม (delivery layer) หน้าที่คือ
// "แปลง HTTP request ให้เป็นเรียกใช้ business logic แล้วแปลงผลลัพธ์กลับเป็น HTTP response"
// มันไม่คุยกับ DB เองเลย มีแค่ noteUsecase (interface จาก domain layer) ไว้เรียกใช้
type NoteHandler struct {
	noteUsecase domain.NoteUsecase
}

// NewNoteHandler คือ constructor รับ nu (ของจริงคือ *usecase.noteUsecase จาก note_usecase.go)
// เข้ามาทาง parameter type domain.NoteUsecase (dependency injection เหมือน layer อื่นๆ)
func NewNoteHandler(nu domain.NoteUsecase) *NoteHandler {
	return &NoteHandler{noteUsecase: nu}
}

// createNoteRequest คือ struct ไว้ decode JSON body ที่ client ส่งเข้ามาตอนสร้าง/แก้ไขโน้ต
// มี json tag กำกับไว้เพื่อบอกว่า key ใน JSON ชื่ออะไร map กับ field ไหน
type createNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Validate เช็คกฎของ field ระดับ "รูปแบบ request" (ไม่ใช่ business rule ซึ่งควรอยู่ชั้น usecase)
// trim ค่า Title/Content ให้เลยในตัว (mutate ผ่าน pointer receiver) เพื่อไม่ให้เก็บ
// ช่องว่างหัว-ท้ายที่ไม่มีความหมายลง DB แล้วคืน map ของ field -> ข้อความ error (ว่างแปลว่าผ่านหมด)
//
// ใช้เป็นแนวทางสำหรับ request struct ของ resource อื่นในอนาคต: เพิ่มเมธอด Validate()
// ให้ DTO ของตัวเอง แล้วเรียกใน handler ก่อนส่งต่อไปชั้น usecase
func (req *createNoteRequest) Validate() map[string]string {
	errs := map[string]string{}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		errs["title"] = "title is required"
	} else if len(req.Title) > noteTitleMaxLen {
		errs["title"] = "title must be at most 255 characters"
	}

	req.Content = strings.TrimSpace(req.Content)
	if len(req.Content) > noteContentMaxLen {
		errs["content"] = "content must be at most 10,000 characters"
	}

	return errs
}

// Create = handler ของ endpoint "สร้างโน้ตใหม่" (เช่น POST /notes)
// w = ตัวเขียน response กลับไปหา client, r = ข้อมูล request ที่เข้ามา
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	// middleware.GetUserFromContext ดึงข้อมูล user ที่ล็อกอินอยู่ ออกจาก context ของ request
	// (context ถูกยัดค่านี้ไว้ตั้งแต่ชั้น middleware auth ก่อนจะส่งต่อมาถึง handler นี้)
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		// ไม่มี user แปลว่ายังไม่ได้ login หรือ token ไม่ถูกต้อง ตอบ 401 แล้วจบทันที (return)
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// เตรียมตัวแปรว่างไว้รับค่า แล้วแปลง (decode) JSON จาก r.Body ใส่เข้าไปใน req
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// body ไม่ใช่ JSON ที่ถูกต้อง หรือ field ผิด type ตอบ 400 Bad Request
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// เช็ค required/ความยาว ก่อนส่งต่อไปชั้น usecase (ดู Validate() ด้านบน)
	if errs := req.Validate(); len(errs) > 0 {
		response.ValidationErrors(w, errs)
		return
	}

	// ประกอบร่าง domain.Note ขึ้นมาจากข้อมูลที่ได้ (UserID เอามาจาก user ที่ login ไม่ใช่จาก client
	// ส่งมาเอง เพื่อกันไม่ให้ client ปลอมตัวสร้างโน้ตแทนคนอื่น)
	note := &domain.Note{
		UserID:  user.ID,
		Title:   req.Title,
		Content: req.Content,
	}

	// เรียก business logic ชั้น usecase (h.noteUsecase.Create ใน note_usecase.go)
	// ซึ่งจะไปเรียก noteRepo.Create ต่อ (ใน note_repo.go) เพื่อ insert ลง Postgres จริง
	if err := h.noteUsecase.Create(r.Context(), note); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create note")
		return
	}

	// สำเร็จ: ตอบ 201 Created พร้อม note ที่ถูกเติม ID/CreatedAt/UpdatedAt มาแล้วจากชั้น repository
	response.JSON(w, http.StatusCreated, note)
}

// GetByID = handler ของ endpoint "ดึงโน้ต 1 รายการ" (เช่น GET /notes/{id})
func (h *NoteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// r.PathValue("id") ดึงค่า {id} จาก URL path (เช่น /notes/42 -> "42")
	// ต้องใช้ Go 1.22+ ที่ net/http รองรับ path parameter แบบนี้เอง
	idStr := r.PathValue("id")
	// id ที่ได้จาก URL เป็น string เสมอ ต้องแปลงเป็น int ก่อนเอาไปใช้เทียบกับ DB
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid note ID")
		return
	}

	// เรียก usecase.GetByID -> repo.GetByID (ใน note_repo.go) ไล่ query SELECT ... WHERE id AND user_id
	note, err := h.noteUsecase.GetByID(r.Context(), id, user.ID)
	if err != nil {
		// errors.Is เช็คว่า err ที่ได้กลับมา "คือ" domain.ErrNotFound หรือเปล่า
		// (ค่านี้ถูก set มาจากชั้น repository ตอนที่ query ไม่เจอแถวไหนเลย)
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Note not found")
			return
		}
		// error อื่นๆ ที่ไม่ใช่ "ไม่เจอ" (เช่น DB ล่ม) ถือเป็น 500
		response.Error(w, http.StatusInternalServerError, "Failed to get note")
		return
	}

	response.JSON(w, http.StatusOK, note)
}

// List = handler ของ endpoint "ดึงโน้ตทั้งหมดของ user คนที่ login อยู่" (เช่น GET /notes)
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// เรียก usecase.List -> repo.ListByUserID (ใน note_repo.go)
	notes, err := h.noteUsecase.List(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list notes")
		return
	}

	// return empty array instead of null
	// ถ้า user ไม่มีโน้ตเลย ชั้น repository จะคืน notes เป็น nil slice
	// ถ้าปล่อยให้ encode เป็น JSON ตรงๆ จะได้ "null" แทนที่จะเป็น "[]" ซึ่ง client (เช่น frontend)
	// ส่วนใหญ่คาดหวัง array เปล่าๆ มากกว่า null จึงต้องเช็คแล้วแทนที่ด้วย slice ว่างก่อนตอบกลับ
	if notes == nil {
		notes = []*domain.Note{}
	}

	response.JSON(w, http.StatusOK, notes)
}

// Update = handler ของ endpoint "แก้ไขโน้ต" (เช่น PUT/PATCH /notes/{id})
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid note ID")
		return
	}

	// ใช้ struct createNoteRequest ตัวเดียวกับตอน Create เพราะ field ที่แก้ไขได้เหมือนกัน
	// (title, content) ไม่จำเป็นต้องสร้าง struct request แยกอีกตัว
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		response.ValidationErrors(w, errs)
		return
	}

	// ประกอบ note ที่จะเอาไปอัปเดต ใส่ ID (จาก URL) และ UserID (จาก user ที่ login)
	// ไว้เป็นเงื่อนไขด้วย เพื่อกันไม่ให้แก้ไขโน้ตของ user คนอื่นได้
	note := &domain.Note{
		ID:      id,
		UserID:  user.ID,
		Title:   req.Title,
		Content: req.Content,
	}

	// เรียก usecase.Update -> repo.Update (ใน note_repo.go)
	// repository เป็นคนเช็คเองว่า id + user_id นี้ match แถวไหนไหม ถ้าไม่ match จะได้ ErrNotFound กลับมา
	if err := h.noteUsecase.Update(r.Context(), note); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Note not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to update note")
		return
	}

	// fetch updated note to return it
	// ตอน Update เราส่งแค่ title/content เข้าไป ไม่มี created_at ติดมาด้วย (note ในตัวแปรข้างบน
	// จึงมีข้อมูลไม่ครบ) เลยต้องยิง GetByID ซ้ำอีกรอบ เพื่อได้ข้อมูลเต็มๆ (รวม updated_at ล่าสุด)
	// กลับไปตอบ client แทน — จุดนี้ error จาก GetByID ถูก "ละเว้น" ด้วย _ เพราะถือว่า Update
	// สำเร็จไปแล้วในขั้นก่อนหน้า ต่อให้ fetch รอบสองพลาดก็ไม่ถือเป็นความผิดพลาดร้ายแรง
	updatedNote, _ := h.noteUsecase.GetByID(r.Context(), id, user.ID)
	response.JSON(w, http.StatusOK, updatedNote)
}

// Delete = handler ของ endpoint "ลบโน้ต" (เช่น DELETE /notes/{id})
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid note ID")
		return
	}

	// เรียก usecase.Delete -> repo.Delete (ใน note_repo.go) ซึ่งเช็ค RowsAffected() เอง
	// ถ้าไม่มีแถวไหนถูกลบ (ไม่เจอ หรือไม่ใช่เจ้าของ) จะได้ domain.ErrNotFound กลับมา
	if err := h.noteUsecase.Delete(r.Context(), id, user.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "Note not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to delete note")
		return
	}

	// ลบสำเร็จ ไม่มี resource เหลือให้ส่งกลับ จึงตอบแค่ข้อความยืนยันธรรมดา
	response.JSON(w, http.StatusOK, map[string]string{"message": "Note deleted successfully"})
}
