package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/labstack/echo/v5"

	"github.com/bogeia/echo-layout/internal/biz"
	"github.com/bogeia/echo-layout/internal/data"
	"github.com/bogeia/echo-layout/internal/server"
	"github.com/bogeia/echo-layout/internal/service"
)

type failingUserRepository struct{ err error }

func (r failingUserRepository) Create(context.Context, biz.User) error { return r.err }
func (r failingUserRepository) Get(context.Context, string) (*biz.User, error) {
	return nil, r.err
}
func (r failingUserRepository) List(context.Context) ([]biz.User, error) { return nil, r.err }
func (r failingUserRepository) Update(context.Context, biz.User) error   { return r.err }
func (r failingUserRepository) Delete(context.Context, string) error     { return r.err }

func TestHumaRepositoryErrorsAreLogged(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	repo := failingUserRepository{err: errors.New("private repository failure")}
	router := server.NewRouter(logger, service.NewUsers(biz.NewUsers(repo), logger))
	for _, test := range []struct{ method, path, body string }{
		{"POST", "/users", `{"name":"Alice"}`},
		{"GET", "/users/example", ""},
		{"GET", "/users", ""},
		{"PUT", "/users/example", `{"name":"Bob"}`},
		{"DELETE", "/users/example", ""},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			logs.Reset()
			request := httptest.NewRequestWithContext(t.Context(), test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			var problem huma.ErrorModel
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusInternalServerError || problem.Status != http.StatusInternalServerError || problem.Detail != "internal server error" {
				t.Fatalf("unexpected response: %d %s", response.Code, response.Body)
			}
			if strings.Contains(response.Body.String(), repo.err.Error()) {
				t.Fatal("response leaked repository error")
			}
			internalErrors := 0
			for line := range strings.SplitSeq(strings.TrimSpace(logs.String()), "\n") {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["msg"] == "internal error" {
					internalErrors++
					if record["level"] != "ERROR" || record["error"] != repo.err.Error() {
						t.Fatalf("original error missing from log: %s", line)
					}
				}
			}
			if internalErrors != 1 {
				t.Fatalf("expected one internal error log, got %d: %s", internalErrors, &logs)
			}
		})
	}
}

func TestCRUDAndOpenAPI(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	store, cleanup, err := data.NewData(logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	router := server.NewRouter(logger, service.NewUsers(biz.NewUsers(data.NewUsers(store, logger)), logger))
	router.GET("/panic", func(*echo.Context) error { panic("private panic detail") })
	router.GET("/error", func(*echo.Context) error { return errors.New("private error detail") })
	router.GET("/committed", func(c *echo.Context) error {
		if err := c.String(http.StatusOK, "already written"); err != nil {
			return err
		}
		return errors.New("error after response")
	})
	for _, test := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/users", `{"name":"Alice"}`, 201},
		{"GET", "/users", "", 200},
		{"GET", "/users/example", "", 404},
		{"PUT", "/users/example", `{"name":"Bob"}`, 200},
		{"DELETE", "/users/example", "", 204},
		{"POST", "/users", `{}`, 422},
		{"POST", "/users", `{"name":"  "}`, 422},
		{"POST", "/users", `{"name":1}`, 422},
		{"POST", "/users", `{`, 400},
		{"GET", "/docs", "", 200},
		{"GET", "/openapi.json", "", 200},
		{"GET", "/openapi.yaml", "", 200},
		{"GET", "/missing", "", 404},
		{"PATCH", "/users", "", 405},
		{"HEAD", "/missing", "", 404},
		{"GET", "/panic", "", 500},
		{"GET", "/error", "", 500},
		{"GET", "/committed", "", 200},
	} {
		t.Run(test.method+test.path+test.body, func(t *testing.T) {
			defer func() {
				if value := recover(); value != nil {
					t.Errorf("panic escaped HTTP handler: %v", value)
				}
			}()
			logStart := logs.Len()
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d, want %d: %s", response.Code, test.status, response.Body)
			}
			if test.status >= 400 {
				if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/problem+json") {
					t.Errorf("unexpected error content type: %s", contentType)
				}
				if test.method != "HEAD" {
					var problem huma.ErrorModel
					if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil || problem.Status != test.status || problem.Title == "" {
						t.Errorf("invalid problem response: %s", response.Body)
					}
				}
				if strings.Contains(response.Body.String(), "private") {
					t.Error("internal error details leaked to client")
				}
			}
			if test.method == "HEAD" && response.Body.Len() != 0 {
				t.Error("HEAD response contains a body")
			}
			if test.status == 405 && response.Header().Get("Allow") == "" {
				t.Error("405 response lost Allow header")
			}
			if test.path == "/committed" && response.Body.String() != "already written" {
				t.Errorf("error handler rewrote committed response: %s", response.Body)
			}
			if test.method == "GET" && test.path == "/users" && strings.TrimSpace(response.Body.String()) != "[]" {
				t.Fatalf("list must be an empty array: %s", response.Body)
			}
			if test.status == 201 || test.method == "PUT" {
				var user service.User
				if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil || user.ID == "" || user.Name == "" {
					t.Fatalf("invalid user response: %s", response.Body)
				}
			}
			accessLogs := 0
			for line := range strings.SplitSeq(strings.TrimSpace(logs.String()[logStart:]), "\n") {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Errorf("invalid log: %v", err)
					continue
				}
				if record["msg"] != "http request" {
					continue
				}
				accessLogs++
				if record["method"] != test.method || record["path"] != test.path || record["status"] != float64(test.status) || record["latency"] == nil {
					t.Errorf("invalid access log: %s", line)
				}
				if test.status == 500 && (record["level"] != "ERROR" || record["error"] == nil) {
					t.Error("server error missing error log detail")
				}
			}
			if accessLogs != 1 {
				t.Errorf("expected one access log, got %d", accessLogs)
			}
		})
	}
	if lines := bytes.Count(logs.Bytes(), []byte(`"msg":"data.users.`)); lines != 5 {
		t.Fatalf("expected one data log per valid CRUD call, got %d: %s", lines, &logs)
	}
	// Routes are registered without logging.Handler, so no input is logged.
	if bytes.Contains(logs.Bytes(), []byte(`"msg":"request args"`)) {
		t.Fatalf("input logging must be opt-in: %s", &logs)
	}
}
