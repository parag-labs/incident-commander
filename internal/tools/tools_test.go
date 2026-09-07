package tools

import (
	"context"
	"testing"
	"time"

	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/pkg/models"
)

func fixedNow() time.Time { return time.Unix(0, 0) }

func counter() func() string {
	n := 0
	return func() string { n++; return "e" + itoa(n) }
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestInvestigatorProducesGroundedEvidence(t *testing.T) {
	sc, _ := simulator.ScenarioByID("database_overload")
	env := simulator.New(sc)
	r := NewRegistry()
	ev, call, err := r.Collect(context.Background(), env, "get_database_health", models.ServiceDatabase, counter(), fixedNow)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if len(ev) != 1 || ev[0].ID == "" {
		t.Fatalf("expected one evidence item with an ID, got %+v", ev)
	}
	if call.Safety != models.SafetyLow || !call.OK {
		t.Fatalf("investigation call should be low-safety and OK, got %+v", call)
	}
	if ev[0].Detail["db_connections"].(int) != 100 {
		t.Fatalf("evidence should reflect the degraded db, got %+v", ev[0].Detail)
	}
}

func TestUnknownToolIsRejected(t *testing.T) {
	env := simulator.New(simulator.Scenarios[0])
	r := NewRegistry()
	_, _, err := r.Collect(context.Background(), env, "totally_made_up_tool", models.ServiceAPI, counter(), fixedNow)
	if _, ok := err.(ErrUnknownTool); !ok {
		t.Fatalf("want ErrUnknownTool, got %v", err)
	}
}

func TestHighSafetyToolNeverAutoAuthorized(t *testing.T) {
	sc, _ := simulator.ScenarioByID("deployment_regression")
	env := simulator.New(sc)
	r := NewRegistry()
	// Even with medium auto-approval on, HIGH must never run automatically.
	_, call, err := r.Remediate(context.Background(), env, "rollback_deployment", models.ServiceAPI,
		Capabilities{AllowAutoMedium: true}, fixedNow)
	if _, ok := err.(ErrUnauthorized); !ok {
		t.Fatalf("want ErrUnauthorized for a HIGH tool, got %v", err)
	}
	if call.OK {
		t.Fatal("unauthorized call should not be marked OK")
	}
	if env.Fixed() {
		t.Fatal("environment must not change when a tool is unauthorized")
	}
}

func TestMediumToolNeedsAutoApproval(t *testing.T) {
	sc, _ := simulator.ScenarioByID("cache_failure")
	env := simulator.New(sc)
	r := NewRegistry()

	// Without auto-approval, a MEDIUM tool is refused.
	if _, _, err := r.Remediate(context.Background(), env, "clear_cache", models.ServiceCache, Capabilities{}, fixedNow); err == nil {
		t.Fatal("MEDIUM tool should be refused without auto-approval")
	}
	// With it, the tool runs and fixes the fault.
	improved, call, err := r.Remediate(context.Background(), env, "clear_cache", models.ServiceCache,
		Capabilities{AllowAutoMedium: true}, fixedNow)
	if err != nil || !improved || !call.OK {
		t.Fatalf("MEDIUM tool with approval should run and improve, got improved=%v err=%v", improved, err)
	}
}

func TestCancelledContextAbortsCollection(t *testing.T) {
	env := simulator.New(simulator.Scenarios[0])
	r := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := r.Collect(ctx, env, "query_metrics", models.ServiceAPI, counter(), fixedNow)
	if err == nil {
		t.Fatal("cancelled context should abort collection")
	}
}
