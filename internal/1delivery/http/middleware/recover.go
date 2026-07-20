package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"go-minimal-backend/pkg/reqctx"
	"go-minimal-backend/pkg/response"
)

// Recover ดัก panic ที่เกิดขึ้นระหว่าง handler ทำงาน (เช่น nil pointer, index out of range)
// ปกติ net/http จะ recover panic ให้เองอยู่แล้วต่อ connection แต่แค่ปิด connection เฉยๆ
// โดยไม่ส่ง response กลับไปหา client เลย middleware นี้ทำให้ client ได้ JSON 500 พร้อม
// request_id เหมือน error ปกติทุกจุด และ log stack trace ไว้ฝั่ง server ให้ debug ได้ทันที
//
// ต้องอยู่ "ใน" RequestID แต่ "นอก" ตัว handler จริง (ดู router.go) เพื่อให้ตอน panic
// ยังอ่าน request_id จาก context ได้ (RequestID ต้องตั้งค่าไปแล้วก่อนถึง handler)
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[%s] panic recovered: %v\n%s", reqctx.RequestID(r.Context()), rec, debug.Stack())
				response.Error(w, r, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
