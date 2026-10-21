package httpapi_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/httpapi"
)

func TestWriteJSON_SetsContentTypeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()

	httpapi.WriteJSON(rec, 201, map[string]string{"hello": "world"})

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["hello"] != "world" {
		t.Fatalf("body = %+v, want hello=world", body)
	}
}

func TestWriteError_ProducesConsistentShape(t *testing.T) {
	rec := httptest.NewRecorder()

	httpapi.WriteError(rec, 400, "bad_request", "name is required")

	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var body httpapi.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Fatalf("error.code = %q, want bad_request", body.Error.Code)
	}
	if body.Error.Message != "name is required" {
		t.Fatalf("error.message = %q, want %q", body.Error.Message, "name is required")
	}
}
