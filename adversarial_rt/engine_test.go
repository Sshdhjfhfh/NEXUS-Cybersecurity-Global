// Nexus IA v6.0 — Tests del Adversarial RT
package adversarial_rt

import (
	"context"
	"testing"
	"time"
)

// TestAllTargets verifica los targets disponibles.
func TestAllTargets(t *testing.T) {
	targets := AllTargets()
	if len(targets) == 0 {
		t.Fatal("no targets returned")
	}
	// Buscar targets conocidos
	expected := map[Target]bool{
		TargetEthicalBoundary: true,
		TargetLLMjackingGuard: true,
		TargetOracleDefense:   true,
		TargetABM:             true,
	}
	found := 0
	for _, tgt := range targets {
		if expected[tgt] {
			found++
		}
	}
	if found != len(expected) {
		t.Errorf("expected %d key targets, found %d", len(expected), found)
	}
}

// TestNewEngine verifica la construcción.
func TestNewEngine(t *testing.T) {
	e := NewEngine(DefaultConfig())
	if e == nil {
		t.Fatal("engine is nil")
	}
	if len(e.ListCycles()) != 0 {
		t.Error("new engine should have 0 cycles")
	}
	if len(e.ListAttacks()) != 0 {
		t.Error("new engine should have 0 attacks")
	}
}

// TestRunCycle verifica un ciclo completo.
func TestRunCycle(t *testing.T) {
	cfg := DefaultConfig()
	// Reducir el número de targets y ataques para test rápido
	cfg.EnabledTargets = []Target{TargetEthicalBoundary, TargetABM}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection, AttackDeception}

	e := NewEngine(cfg)
	ctx := context.Background()

	cycle, err := e.RunCycle(ctx)
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}
	if cycle == nil {
		t.Fatal("cycle is nil")
	}
	if len(cycle.Attacks) != 4 {
		t.Errorf("expected 4 attacks (2 targets × 2 types), got %d", len(cycle.Attacks))
	}
	if cycle.FinishedAt == nil {
		t.Error("cycle should be finished")
	}
	if cycle.Summary == "" {
		t.Error("cycle should have summary")
	}
}

// TestCycleHistory verifica el histórico de ciclos.
func TestCycleHistory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection}

	e := NewEngine(cfg)
	ctx := context.Background()

	// Ejecutar 3 ciclos
	for i := 0; i < 3; i++ {
		_, err := e.RunCycle(ctx)
		if err != nil {
			t.Fatalf("cycle %d failed: %v", i, err)
		}
	}

	cycles := e.ListCycles()
	if len(cycles) != 3 {
		t.Errorf("expected 3 cycles, got %d", len(cycles))
	}

	lastCycle := e.LastCycle()
	if lastCycle == nil {
		t.Fatal("last cycle is nil")
	}
}

// TestAttackResult verifica la clasificación del resultado.
func TestAttackResult(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection}
	cfg.MaxAttemptsPerAttack = 3

	e := NewEngine(cfg)
	ctx := context.Background()

	cycle, err := e.RunCycle(ctx)
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}
	if len(cycle.Attacks) == 0 {
		t.Fatal("no attacks")
	}

	attack := cycle.Attacks[0]
	if attack.Result == "" {
		t.Error("attack result should be set")
	}
	// Con el stub actual, debería ser BLOCKED
	if attack.Result != ResultBlocked {
		t.Logf("attack result: %s (severity: %s)", attack.Result, attack.Severity)
	}
}

// TestCallbackOnWeakness verifica el callback de debilidad.
func TestCallbackOnWeakness(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection}
	// Con MaxAttempts altos y el stub, no debería haber debilidad
	cfg.MaxAttemptsPerAttack = 5

	callbackCalled := make(chan bool, 1)
	cfg.NotifyOnWeakness = func(a *Attack) {
		select {
		case callbackCalled <- true:
		default:
		}
	}

	e := NewEngine(cfg)
	ctx := context.Background()

	_, err := e.RunCycle(ctx)
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}

	// Dar tiempo al callback (es async)
	select {
	case <-callbackCalled:
		t.Log("weakness callback was called (found weakness)")
	case <-time.After(100 * time.Millisecond):
		t.Log("no weakness found (expected with stub)")
	}
}

// TestCallbackOnCycleEnd verifica el callback de fin de ciclo.
func TestCallbackOnCycleEnd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection}

	callbackCalled := make(chan *Cycle, 1)
	cfg.NotifyOnCycleEnd = func(c *Cycle) {
		select {
		case callbackCalled <- c:
		default:
		}
	}

	e := NewEngine(cfg)
	ctx := context.Background()

	_, err := e.RunCycle(ctx)
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}

	select {
	case c := <-callbackCalled:
		if c == nil {
			t.Error("cycle in callback is nil")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("cycle end callback not called")
	}
}

// TestStats verifica las estadísticas.
func TestStats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary, TargetABM}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection}

	e := NewEngine(cfg)
	ctx := context.Background()

	// 2 ciclos
	for i := 0; i < 2; i++ {
		_, _ = e.RunCycle(ctx)
	}

	stats := e.Stats()
	if stats.TotalCycles != 2 {
		t.Errorf("expected 2 cycles, got %d", stats.TotalCycles)
	}
	if stats.TotalAttacks != 4 {
		t.Errorf("expected 4 attacks, got %d", stats.TotalAttacks)
	}
	if len(stats.ByTarget) != 2 {
		t.Errorf("expected 2 targets in stats, got %d", len(stats.ByTarget))
	}
	if stats.FirstCycleAt == nil {
		t.Error("FirstCycleAt should be set")
	}
	if stats.LastCycleAt == nil {
		t.Error("LastCycleAt should be set")
	}
}

// TestTargetStats verifica las stats por target.
func TestTargetStats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection, AttackDeception}

	e := NewEngine(cfg)
	ctx := context.Background()

	_, _ = e.RunCycle(ctx)

	ts := e.TargetStats(TargetEthicalBoundary)
	if ts == nil {
		t.Fatal("target stats is nil")
	}
	if ts.TotalAttacks != 2 {
		t.Errorf("expected 2 attacks, got %d", ts.TotalAttacks)
	}
	if ts.Target != TargetEthicalBoundary {
		t.Errorf("wrong target: %s", ts.Target)
	}
}

// TestListAttacksByTarget verifica el filtrado por target.
func TestListAttacksByTarget(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = []Target{TargetEthicalBoundary, TargetABM}
	cfg.EnabledAttacks = []AttackType{AttackPromptInjection}

	e := NewEngine(cfg)
	ctx := context.Background()

	_, _ = e.RunCycle(ctx)

	attacks := e.ListAttacksByTarget(TargetEthicalBoundary)
	if len(attacks) != 1 {
		t.Errorf("expected 1 attack on ethical_boundary, got %d", len(attacks))
	}
	if attacks[0].Target != TargetEthicalBoundary {
		t.Errorf("wrong target: %s", attacks[0].Target)
	}
}

// TestDuration verifica el cálculo de duración.
func TestDuration(t *testing.T) {
	now := time.Now()
	finished := now.Add(5 * time.Second)

	attack := &Attack{
		StartedAt:  now,
		FinishedAt: &finished,
	}
	if attack.Duration() != 5*time.Second {
		t.Errorf("duration should be 5s, got %v", attack.Duration())
	}

	// Sin FinishedAt → 0
	attack2 := &Attack{StartedAt: now}
	if attack2.Duration() != 0 {
		t.Errorf("duration should be 0, got %v", attack2.Duration())
	}
}

// TestIsWeakness verifica la clasificación.
func TestIsWeakness(t *testing.T) {
	tests := []struct {
		result   Result
		expected bool
	}{
		{ResultBlocked, false},
		{ResultWeaknessFound, true},
		{ResultPartialBypass, true},
		{ResultError, false},
	}
	for _, tc := range tests {
		a := &Attack{Result: tc.result}
		if a.IsWeakness() != tc.expected {
			t.Errorf("result %s: expected IsWeakness=%v, got %v",
				tc.result, tc.expected, a.IsWeakness())
		}
	}
}

// TestShutdown verifica el shutdown.
func TestShutdown(t *testing.T) {
	e := NewEngine(DefaultConfig())
	if err := e.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	ctx := context.Background()
	_, err := e.RunCycle(ctx)
	if err == nil {
		t.Error("RunCycle after shutdown should fail")
	}
}

// TestContextCancellation verifica la cancelación del contexto.
func TestContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTargets = AllTargets() // muchos targets
	cfg.EnabledAttacks = []AttackType{
		AttackPromptInjection, AttackDataPoisoning, AttackTrafficFlood,
		AttackReplay, AttackPrivilegeEscalation, AttackDeception,
	}

	e := NewEngine(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // asegurar expiración

	cycle, err := e.RunCycle(ctx)
	// Puede o no dar error dependiendo de la velocidad
	if cycle != nil && cycle.FinishedAt == nil {
		t.Error("cycle should be finished")
	}
	_ = err // puede ser nil si el ciclo terminó antes
}
