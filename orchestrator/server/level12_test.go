package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nexus-ia/nexus/db"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/lab/network"
	"github.com/nexus-ia/nexus/orchestrator/level12"
	"github.com/nexus-ia/nexus/swarm"
)

func TestLabHTTPGuardsEvidencePersistenceAndFeedback(t *testing.T) {
	s, f, request := trafficEvidenceServer(t)
	finding, err := s.m.pg.GetFinding(f.FindingID)
	if err != nil {
		t.Fatal(err)
	}
	if finding.TaskID == nil {
		t.Fatal("fixture missing task")
	}
	taskID := *finding.TaskID
	base := fmt.Sprintf("/api/lab/tasks/%d", taskID)
	unauth := httptest.NewRecorder()
	s.Handler().ServeHTTP(unauth, httptest.NewRequest("GET", "/api/lab/status", nil))
	if unauth.Code != 401 {
		t.Fatalf("unauthenticated: %d", unauth.Code)
	}
	response := request("GET", "/api/lab/status", "")
	var state level12.Status
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &state) != nil || state.Ready {
		t.Fatal("default gate is open", response.Body.String())
	}
	body := `{"target":"10.20.0.3","evidence_ids":["lab-capture"]}`
	if w := request("POST", base+"/runs", body); w.Code != 503 {
		t.Fatalf("missing guards: %d %s", w.Code, w.Body.String())
	}
	scope, err := network.NewScope("10.20.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	allow := func(context.Context, lab.Action) error { return nil }
	gate, err := lab.NewGate(scope, lab.Checks{Identity: allow, OPSEC: allow, Ethics: allow, Monitor: allow})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[swarm.Role]string{}
	for _, role := range swarm.Roles() {
		ids[role] = "test-" + string(role)
	}
	if err := s.ConfigureLevel12(level12.Config{Gate: gate, AgentIDs: ids}); err != nil {
		t.Fatal(err)
	}
	if w := request("POST", base+"/runs", body); w.Code != 403 {
		t.Fatalf("unowned host: %d %s", w.Code, w.Body.String())
	}
	_, err = s.m.pg.Assets().UpsertIP(db.UpsertIPReq{IP: "10.20.0.3", TaskID: taskID})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.m.pg.Exec(`DELETE FROM assets WHERE ip='10.20.0.3' AND $1=ANY(task_ids)`, taskID) })
	seedServerEvidenceFlow(t, s, "lab-capture", []byte("credentials stay outside reports"))
	_, err = s.m.traffic.DB().Exec(`UPDATE exchanges SET url='http://10.20.0.3/',host='10.20.0.3' WHERE id='lab-capture'`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.m.traffic.DB().Exec(`UPDATE exchange_bodies SET resp_head=? WHERE id='lab-capture'`, "HTTP 200\nContent-Type: text/html\nSet-Cookie: session=secret\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{body + ` {}`, strings.Replace(body, `"target"`, `"unknown"`, 1)} {
		if w := request("POST", base+"/runs", bad); w.Code != 400 {
			t.Fatalf("invalid JSON accepted: %d", w.Code)
		}
	}
	if w := request("POST", base+"/runs", `{"target":"`+strings.Repeat("a", 33000)+`"}`); w.Code != 413 {
		t.Fatalf("oversize body: %d", w.Code)
	}
	w := request("POST", base+"/runs", body)
	if w.Code != 201 {
		t.Fatalf("analysis: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "credentials") {
		t.Fatal("raw evidence leaked")
	}
	var run level12.Run
	if err := json.Unmarshal(w.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if len(run.Workers) != 8 || len(run.Findings) < 3 {
		t.Fatal("incomplete pipeline")
	}
	if w := request("POST", base+"/runs", body); w.Code != 409 {
		t.Fatalf("replayed evidence: %d %s", w.Code, w.Body.String())
	}
	if w := request("GET", base+"/runs/"+run.ID, ""); w.Code != 200 {
		t.Fatal("report not readable", w.Body.String())
	}
	if w := request("GET", base+"/runs?limit=101", ""); w.Code != 400 {
		t.Fatal("unbounded history")
	}
	feedback := base + "/runs/" + run.ID + "/feedback"
	if w := request("POST", feedback, `{"class":"cleartext","evidence_id":"lab-capture"}`); w.Code != 400 {
		t.Fatal("missing explicit evaluation accepted")
	}
	value := `{"class":"cleartext","evidence_id":"lab-capture","confirmed":true}`
	if w := request("POST", feedback, value); w.Code != 200 {
		t.Fatal("feedback failed", w.Body.String())
	}
	if w := request("POST", feedback, value); w.Code != 409 {
		t.Fatal("duplicate evaluation accepted")
	}
	// Stored evidence for another address cannot be imported by choosing its ID.
	seedServerEvidenceFlow(t, s, "other-target", nil)
	if w := request("POST", base+"/runs", `{"target":"10.20.0.3","evidence_ids":["other-target"]}`); w.Code != 400 {
		t.Fatalf("mismatched evidence: %d %s", w.Code, w.Body.String())
	}
	if err := s.ConfigureLevel12(level12.Config{}); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", base+"/runs", ""); w.Code != 200 || !strings.Contains(w.Body.String(), run.ID) {
		t.Fatal("history blocked without execution guards")
	}
	if w := request("POST", feedback, value); w.Code != 503 {
		t.Fatal("feedback bypassed guards")
	}
}
func TestLabResponseHeadersRejectIncompleteCaptures(t *testing.T) {
	for _, raw := range []string{"", "HTTP/1.1 200 OK", "Set-Cookie: secret\r\n\r\n"} {
		if _, err := labResponseHeaders(raw); err == nil {
			t.Fatalf("accepted incomplete capture %q", raw)
		}
	}
}
