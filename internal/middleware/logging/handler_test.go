package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/bogeia/echo-layout/internal/middleware/logging"
)

type loggedInput struct {
	ID    string `path:"id"`
	Limit int    `query:"limit" minimum:"1"`
	Body  struct {
		Name string `json:"name" minLength:"1"`
	}
}

type redactedInput struct {
	Body struct {
		Password string `json:"password"`
	}
}

func (i *redactedInput) LogValue() slog.Value {
	return slog.GroupValue(slog.String("password", "[REDACTED]"))
}

func TestRequestArgs(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Test", "1"))
	calls := 0
	operation := huma.Operation{OperationID: "input-test", Method: http.MethodPost, Path: "/input/{id}"}
	huma.Register(api, operation, logging.Handler(logger, operation.OperationID,
		func(_ context.Context, input *loggedInput) (*struct{}, error) {
			calls++
			if input.ID != "example" || input.Limit != 2 || input.Body.Name != "Alice" {
				t.Errorf("unexpected handler input: %+v", input)
			}
			if !strings.Contains(logs.String(), `"msg":"request args"`) {
				t.Error("arguments were not logged before the handler")
			}
			return nil, nil
		}))
	operation = huma.Operation{OperationID: "redact-test", Method: http.MethodPost, Path: "/redacted"}
	huma.Register(api, operation, logging.Handler(logger, operation.OperationID,
		func(_ context.Context, input *redactedInput) (*struct{}, error) {
			calls++
			if input.Body.Password != "secret-value" {
				t.Error("redaction changed the handler input")
			}
			return nil, nil
		}))
	for _, test := range []struct {
		path, body string
		status     int
		args       string
	}{
		{"/input/example?limit=2", `{"name":"Alice"}`, 204, `{"ID":"example","Limit":2,"Body":{"name":"Alice"}}`},
		{"/input/example?limit=0", `{"name":"Alice"}`, 422, ""},
		{"/input/example?limit=2", `{}`, 422, ""},
		{"/input/example?limit=2", `{`, 400, ""},
		{"/redacted", `{"password":"secret-value"}`, 204, `{"password":"[REDACTED]"}`},
	} {
		t.Run(test.path+test.body, func(t *testing.T) {
			logs.Reset()
			before := calls
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d, body=%s", response.Code, response.Body)
			}
			argsLogs := 0
			for line := range strings.SplitSeq(strings.TrimSpace(logs.String()), "\n") {
				if line == "" {
					continue
				}
				var record struct {
					Message   string          `json:"msg"`
					Level     string          `json:"level"`
					Operation string          `json:"operation"`
					Args      json.RawMessage `json:"args"`
				}
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record.Message != "request args" {
					continue
				}
				argsLogs++
				if string(record.Args) != test.args || record.Operation == "" || record.Level != "DEBUG" {
					t.Errorf("unexpected parameter log: %s", line)
				}
			}
			wantCalls := 0
			if test.args != "" {
				wantCalls = 1
			}
			if argsLogs != wantCalls || calls-before != wantCalls {
				t.Fatalf("args logs=%d, handler calls=%d", argsLogs, calls-before)
			}
			if strings.Contains(logs.String(), "secret-value") {
				t.Fatal("password leaked to logs")
			}
		})
	}
}
