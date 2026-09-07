package simulator

import (
	"testing"

	"github.com/parag-labs/incident-commander/pkg/models"
)

func TestScenariosAreDeterministic(t *testing.T) {
	for _, sc := range Scenarios {
		a := New(sc)
		b := New(sc)
		for _, svc := range models.AllServices {
			if a.Snapshot(svc) != b.Snapshot(svc) {
				t.Fatalf("scenario %s not deterministic for %s", sc.ID, svc)
			}
		}
	}
}

func TestFaultDegradesTheAffectedService(t *testing.T) {
	e := New(mustScenario(t, "database_overload"))
	db := e.Snapshot(models.ServiceDatabase)
	if db.Healthy {
		t.Fatal("database should be unhealthy under database_overload")
	}
	if db.DBConns != 100 || db.LatencyP99MS < 2000 {
		t.Fatalf("expected degraded db metrics, got %+v", db)
	}
	// An unaffected service stays at baseline.
	q := e.Snapshot(models.ServiceQueue)
	if !q.Healthy {
		t.Fatalf("queue should be healthy under database_overload, got %+v", q)
	}
}

func TestCorrectRemediationClearsTheFault(t *testing.T) {
	sc := mustScenario(t, "database_overload")
	e := New(sc)
	if ok := e.Apply(sc.FixAction, sc.FixService); !ok {
		t.Fatal("correct remediation should report success")
	}
	if !e.Fixed() {
		t.Fatal("environment should be marked fixed")
	}
	if !e.Snapshot(models.ServiceDatabase).Healthy {
		t.Fatal("database should be healthy after the correct fix")
	}
}

func TestWrongRemediationDoesNotHelp(t *testing.T) {
	sc := mustScenario(t, "database_overload")
	e := New(sc)
	// Restarting the API does nothing for a database connection leak.
	if ok := e.Apply("restart_service", models.ServiceAPI); ok {
		t.Fatal("wrong remediation should report no improvement")
	}
	if e.Fixed() {
		t.Fatal("environment must stay degraded after a wrong remediation")
	}
	if e.Snapshot(models.ServiceDatabase).Healthy {
		t.Fatal("database should still be unhealthy after a wrong remediation")
	}
}

func TestCascadingFailureRootIsDatabase(t *testing.T) {
	sc := mustScenario(t, "cascading_failure")
	e := New(sc)
	// Fixing the API alone must not resolve a database-rooted cascade.
	if e.Apply("scale_service", models.ServiceAPI) {
		t.Fatal("scaling api should not fix a database-rooted cascade")
	}
	if ok := e.Apply(sc.FixAction, sc.FixService); !ok {
		t.Fatal("fixing the database root should resolve the cascade")
	}
	if !e.Snapshot(models.ServiceAPI).Healthy {
		t.Fatal("api should recover once the database root is fixed")
	}
}

func TestAllTenScenariosPresent(t *testing.T) {
	if len(Scenarios) != 10 {
		t.Fatalf("expected 10 evaluation scenarios, got %d", len(Scenarios))
	}
	seen := map[string]bool{}
	for _, s := range Scenarios {
		if seen[s.ID] {
			t.Fatalf("duplicate scenario id %s", s.ID)
		}
		seen[s.ID] = true
		if s.FixAction == "" || s.FixService == "" {
			t.Fatalf("scenario %s missing ground-truth fix", s.ID)
		}
	}
}

func mustScenario(t *testing.T, id string) Scenario {
	t.Helper()
	s, ok := ScenarioByID(id)
	if !ok {
		t.Fatalf("scenario %s not found", id)
	}
	return s
}
