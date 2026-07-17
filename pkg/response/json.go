package response

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func Error(w http.ResponseWriter, status int, err string) {
	JSON(w, status, ErrorResponse{Error: err})
}

// ValidationErrors ตอบ 400 พร้อม error รายฟิลด์ เช่น {"errors": {"title": "title is required"}}
// ใช้ตอน request ผ่านการ decode JSON ได้ แต่ค่าที่ส่งมาไม่ผ่านกฎ validation (เช่น field required, ความยาวเกิน)
// รูปแบบ per-field แบบนี้ทำให้ frontend เอาไปโชว์ error ใต้ input แต่ละช่องของฟอร์มได้ตรงๆ
func ValidationErrors(w http.ResponseWriter, errors map[string]string) {
	JSON(w, http.StatusBadRequest, map[string]any{"errors": errors})
}
