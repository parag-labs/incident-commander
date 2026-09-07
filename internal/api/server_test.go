package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/parag-labs/incident-commander/pkg/models"
)

func newTestServer() http.Handler {
	s := New(Config{Workers: 4})
	return s.Handler(true, 4)
}

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestCreateIncidentRunsFullPipeline(t *testing.T) {
	h := newTestServer()
	rr := post(t, h, "/incidents", `{"scenario":"database_overload","auto_approve_medium":true}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var inc models.Incident
	if err := json.Unmarshal(rr.Body.Bytes(), &inc); err != nil {
		t.Fatalf("bad response JSON: %v", err)
	}
	if inc.State != models.StateResolved {
		t.Fatalf("want RESOLVED, got %s", inc.State)
	}
	if inc.Remediation == nil || inc.Remediation.Action != "reduce_db_connections" {
		t.Fatalf("unexpected remediation: %+v", inc.Remediation)
	}
}

func TestApprovalFlowOverHTTP(t *testing.T) {
	h := newTestServer()
	// A rollback scenario stops for approval.
	rr := post(t, h, "/incidents", `{"scenario":"deployment_regression","auto_approve_medium":true}`)
	var inc models.Incident
	_ = json.Unmarshal(rr.Body.Bytes(), &inc)
	if inc.State != models.StateWaitingForApproval {
		t.Fatalf("want WAITING_FOR_APPROVAL, got %s", inc.State)
	}
	// Approve it.
	rr2 := post(t, h, "/incidents/"+inc.ID+"/approve", "")
	if rr2.Code != http.StatusOK {
		t.Fatalf("approve want 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var inc2 models.Incident
	_ = json.Unmarshal(rr2.Body.Bytes(), &inc2)
	if inc2.State != models.StateResolved {
		t.Fatalf("after approval want RESOLVED, got %s", inc2.State)
	}
}

func TestUnknownScenarioRejected(t *testing.T) {
	h := newTestServer()
	rr := post(t, h, "/incidents", `{"scenario":"does_not_exist"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestMCPIsReadOnly(t *testing.T) {
	h := newTestServer()
	// Create an incident so there's an environment to query.
	rr := post(t, h, "/incidents", `{"scenario":"cache_failure","auto_approve_medium":true}`)
	var inc models.Incident
	_ = json.Unmarshal(rr.Body.Bytes(), &inc)

	// A read-only method works.
	body := `{"method":"service.health","params":{"incident":"` + inc.ID + `","service":"cache"}}`
	ok := post(t, h, "/mcp", body)
	if ok.Code != http.StatusOK {
		t.Fatalf("read-only MCP method should succeed, got %d: %s", ok.Code, ok.Body.String())
	}
	// A remediation-shaped method is refused - MCP never exposes dangerous ops.
	bad := post(t, h, "/mcp", `{"method":"restart_service","params":{}}`)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("non-read-only MCP method must be refused, got %d", bad.Code)
	}
}

func TestMetricsEndpointExposesCounters(t *testing.T) {
	h := newTestServer()
	_ = post(t, h, "/incidents", `{"scenario":"high_cpu","auto_approve_medium":true}`)
	rr := get(t, h, "/metrics")
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("incident_count")) {
		t.Fatalf("metrics should expose incident_count, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	h := newTestServer()
	if rr := get(t, h, "/healthz"); rr.Code != http.StatusOK {
		t.Fatalf("healthz want 200, got %d", rr.Code)
	}
}
