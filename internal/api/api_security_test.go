package api

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jbutlerdev/tasks/internal/models"
	"github.com/jbutlerdev/tasks/internal/storage"
)

// Regression tests for CodeQL findings in scan/tasks.sarif (2026-10-23 pass):
//   - go/reflected-xss handlers.go:800/970/998 (stored XSS via list.Name, task.Title/Description, task.State)
//   - go/path-injection filestore.go (client-supplied list/task IDs reach filepath.Join)

func newTestRouter(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return NewRouter(store, embed.FS{}), dir
}

// TestStoredXSS_ListName covers POST /api/lists -> GET /lists/{id} HTML.
func TestStoredXSS_ListName(t *testing.T) {
	r, _ := newTestRouter(t)

	payload := `<img src=x onerror=alert(1)>`
	body, _ := json.Marshal(models.TaskList{Name: payload, Description: "<script>alert(2)</script>"})
	req := httptest.NewRequest(http.MethodPost, "/api/lists", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create list: got %d body=%s", rec.Code, rec.Body.String())
	}
	var created models.TaskList
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created list: %v body=%s", err, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/lists/"+created.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get list UI: got %d", rec.Code)
	}
	html := rec.Body.String()
	if strings.Contains(html, payload) {
		t.Errorf("XSS: raw payload %q reflected unescaped into HTML", payload)
	}
	if !strings.Contains(html, "&lt;img") {
		t.Errorf("XSS: payload not HTML-escaped (expected &lt;img)")
	}
	if strings.Contains(html, "<script>alert(2)</script>") {
		t.Errorf("XSS: description script tag reflected unescaped")
	}
}

// TestTaskStateAttrXSS covers unvalidated task.State reaching class/data attributes.
func TestTaskStateAttrXSS(t *testing.T) {
	r, _ := newTestRouter(t)

	// create list
	body, _ := json.Marshal(models.TaskList{Name: "sec"})
	req := httptest.NewRequest(http.MethodPost, "/api/lists", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var created models.TaskList
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode list: %v", err)
	}

	// 1) XSS in state must be REJECTED at the boundary (allowlist).
	statePayload := `todo" onmouseover=alert(1) x="`
	form := "title=hello&state=" + statePayload
	req = httptest.NewRequest(http.MethodPost, "/api/lists/"+created.ID+"/tasks", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("state payload accepted: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 2) legitimate state still works and renders safely.
	form = "title=hello&state=todo"
	req = httptest.NewRequest(http.MethodPost, "/api/lists/"+created.ID+"/tasks", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("legit task create broke: %d %s", rec.Code, rec.Body.String())
	}

	// 3) XSS in task title/description must be stored-but-escaped in UI.
	xssForm := "title=<img src=x onerror=alert(3)>&description=<script>alert(4)</script>&state=todo"
	req = httptest.NewRequest(http.MethodPost, "/api/lists/"+created.ID+"/tasks", strings.NewReader(xssForm))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("xss task create: %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/lists/"+created.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	html := rec.Body.String()
	if strings.Contains(html, "<img src=x onerror=alert(3)>") {
		t.Errorf("XSS: raw title img reflected")
	}
	if strings.Contains(html, "<script>alert(4)</script>") {
		t.Errorf("XSS: raw description script reflected")
	}
	if !strings.Contains(html, "&lt;img") {
		t.Errorf("XSS: title not escaped")
	}
}

func TestListIDPathTraversal(t *testing.T) {
	r, baseDir := newTestRouter(t)

	// 1) create with traversal ID must be rejected
	body, _ := json.Marshal(models.TaskList{ID: "../../evilpwn", Name: "evil"})
	req := httptest.NewRequest(http.MethodPost, "/api/lists", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code < 400 {
		t.Errorf("traversal list ID accepted: status=%d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "..", "..", "evilpwn")); err == nil {
		t.Errorf("traversal: directory created outside baseDir")
	}

	// 2) read-side traversal on GET /api/lists/{id} must be rejected, not 500/ok
	req = httptest.NewRequest(http.MethodGet, "/api/lists/%2e%2e%2f%2e%2e%2fetc", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Errorf("read traversal accepted: %s", rec.Body.String())
	}

	// 3) legitimate uuid IDs keep working
	body, _ = json.Marshal(models.TaskList{Name: "ok-list"})
	req = httptest.NewRequest(http.MethodPost, "/api/lists", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("legit create broke: %d %s", rec.Code, rec.Body.String())
	}
}
