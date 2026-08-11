package toolsinfra_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	toolsinfra "github.com/coderoycc/ai-go-api/internal/infrastructure/tools"
	"github.com/stretchr/testify/assert"
)

type DummyTool struct {
	toolsinfra.BaseHTTPTool
	url string
}

func (d *DummyTool) Name() string                     { return "dummy_tool" }
func (d *DummyTool) Description() string              { return "Dummy tool" }
func (d *DummyTool) Method() string                   { return "POST" }
func (d *DummyTool) EndpointURL() string              { return d.url }
func (d *DummyTool) Parameters() map[string]any       { return nil }
func (d *DummyTool) ResponseSchema() map[string]any   { return nil }
func (d *DummyTool) ExcludedFields() []string         { return []string{"debug_info", "secret_token"} }
func (d *DummyTool) RequiredPermission() models.Permission { return models.PermissionRead }
func (d *DummyTool) FallbackArgKey() string           { return "id" }

func TestBaseHTTPTool_AutomaticExcludedFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "123", "name": "Producto Test", "debug_info": "trace-999", "secret_token": "xyz"}`))
	}))
	defer ts.Close()

	dt := &DummyTool{url: ts.URL}
	dt.BaseHTTPTool = toolsinfra.NewBaseHTTPTool(dt, 5*time.Second)

	res, err := dt.Execute(context.Background(), `{"id": "123"}`)
	assert.NoError(t, err)

	resMap, ok := res.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "123", resMap["id"])
	assert.Equal(t, "Producto Test", resMap["name"])
	assert.Nil(t, resMap["debug_info"])
	assert.Nil(t, resMap["secret_token"])
}
