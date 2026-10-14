// Nexus IA v6.0 — Tests del CTF Engine
package ctf_engine

import (
	"testing"
	"time"
)

// helperChallenge crea un reto de prueba.
func helperChallenge(id string, difficulty Difficulty, category Category) *Challenge {
	return &Challenge{
		ID:          id,
		Name:        "Test " + id,
		Platform:    PlatformLocal,
		Category:    category,
		Difficulty:  difficulty,
		Description: "Test challenge",
		FlagFormat:  "FLAG{...}",
		Points:      100,
		CreatedAt:   time.Now(),
	}
}

// TestLoadChallenge verifica cargar retos.
func TestLoadChallenge(t *testing.T) {
	e := NewEngine(DefaultConfig())

	ch := helperChallenge("ch-1", DifficultyEasy, CategoryWeb)
	if err := e.LoadChallenge(ch); err != nil {
		t.Fatalf("LoadChallenge failed: %v", err)
	}

	if len(e.ListChallenges()) != 1 {
		t.Errorf("expected 1 challenge, got %d", len(e.ListChallenges()))
	}

	// Reto sin ID → error
	if err := e.LoadChallenge(&Challenge{Name: "no-id"}); err == nil {
		t.Error("challenge without ID should fail")
	}

	// Reto nil → error
	if err := e.LoadChallenge(nil); err == nil {
		t.Error("nil challenge should fail")
	}
}

// TestPlatformFilter verifica el filtro de plataforma.
func TestPlatformFilter(t *testing.T) {
	e := NewEngine(DefaultConfig())

	// PlatformLocal está habilitada
	local := helperChallenge("local-1", DifficultyEasy, CategoryWeb)
	if err := e.LoadChallenge(local); err != nil {
		t.Fatalf("local challenge should be accepted: %v", err)
	}

	// PlatformHTB NO está habilitada (solo Local y PicoCTF)
	htb := helperChallenge("htb-1", DifficultyEasy, CategoryWeb)
	htb.Platform = PlatformHTB
	if err := e.LoadChallenge(htb); err == nil {
		t.Error("HTB challenge should be rejected when not enabled")
	}
}

// TestStartFinishAttempt verifica el ciclo de intento.
func TestStartFinishAttempt(t *testing.T) {
	e := NewEngine(DefaultConfig())
	ch := helperChallenge("ch-1", DifficultyEasy, CategoryWeb)
	_ = e.LoadChallenge(ch)

	// Iniciar
	attempt, err := e.StartAttempt("ch-1", "red-agent-01")
	if err != nil {
		t.Fatalf("StartAttempt failed: %v", err)
	}
	if attempt.Solved {
		t.Error("new attempt should not be solved")
	}

	// Sleep corto para que TimeSpent > 0
	time.Sleep(10 * time.Millisecond)

	// Terminar con éxito
	err = e.FinishAttempt(attempt.ID, true, "FLAG{test}", []string{"sql-injection"}, 0)
	if err != nil {
		t.Fatalf("FinishAttempt failed: %v", err)
	}

	// Verificar que el reto está resuelto
	updatedCh, _ := e.GetChallenge("ch-1")
	if !updatedCh.Solved {
		t.Error("challenge should be marked as solved")
	}
	if updatedCh.SolvedBy != "red-agent-01" {
		t.Errorf("wrong solver: %s", updatedCh.SolvedBy)
	}
	if updatedCh.Flag != "FLAG{test}" {
		t.Errorf("wrong flag: %s", updatedCh.Flag)
	}
}

// TestFinishAttemptUnknown verifica errores.
func TestFinishAttemptUnknown(t *testing.T) {
	e := NewEngine(DefaultConfig())

	err := e.FinishAttempt("ghost-attempt", true, "", nil, 0)
	if err == nil {
		t.Error("unknown attempt should fail")
	}
}

// TestMetrics verifica el cálculo de métricas.
func TestMetrics(t *testing.T) {
	e := NewEngine(DefaultConfig())
	ch := helperChallenge("ch-1", DifficultyEasy, CategoryWeb)
	_ = e.LoadChallenge(ch)

	attempt, _ := e.StartAttempt("ch-1", "red-agent-01")
	time.Sleep(10 * time.Millisecond)
	_ = e.FinishAttempt(attempt.ID, true, "FLAG{x}", []string{"xss"}, 0)

	metrics := e.Metrics("ch-1")
	if metrics == nil {
		t.Fatal("metrics should not be nil after solve")
	}
	if metrics.ChallengeID != "ch-1" {
		t.Errorf("wrong challenge: %s", metrics.ChallengeID)
	}
	if metrics.TimeSpent == 0 {
		t.Error("TimeSpent should be > 0")
	}
	// Con tiempo corto (10ms) vs 20min humanos, debería ser percentil ~99%
	if metrics.Percentile < 90 {
		t.Errorf("agent should be ~99%% percentile, got %.2f", metrics.Percentile)
	}
}

// TestRanking verifica el ranking del agente.
func TestRanking(t *testing.T) {
	e := NewEngine(DefaultConfig())

	// 3 retos
	for i := 1; i <= 3; i++ {
		ch := helperChallenge("ch-"+string(rune('0'+i)), DifficultyEasy, CategoryWeb)
		_ = e.LoadChallenge(ch)

		attempt, _ := e.StartAttempt(ch.ID, "agent-01")
		time.Sleep(5 * time.Millisecond)
		_ = e.FinishAttempt(attempt.ID, true, "FLAG{x}", []string{"xss"}, 0)
	}

	ranking := e.Ranking("agent-01")
	if ranking == nil {
		t.Fatal("ranking should not be nil")
	}
	if ranking.TotalSolves != 3 {
		t.Errorf("expected 3 solves, got %d", ranking.TotalSolves)
	}
	if ranking.TotalAttempts != 3 {
		t.Errorf("expected 3 attempts, got %d", ranking.TotalAttempts)
	}
	if ranking.SolveRate != 1.0 {
		t.Errorf("SolveRate should be 1.0, got %.2f", ranking.SolveRate)
	}
	if ranking.EasySolves != 3 {
		t.Errorf("expected 3 easy solves, got %d", ranking.EasySolves)
	}
	if ranking.ByCategory[CategoryWeb] != 3 {
		t.Errorf("expected 3 WEB solves, got %d", ranking.ByCategory[CategoryWeb])
	}
}

// TestRankingPercentile verifica el percentil.
func TestRankingPercentile(t *testing.T) {
	e := NewEngine(DefaultConfig())
	ch := helperChallenge("ch-1", DifficultyEasy, CategoryWeb)
	_ = e.LoadChallenge(ch)

	attempt, _ := e.StartAttempt("ch-1", "agent-01")
	time.Sleep(10 * time.Millisecond)
	_ = e.FinishAttempt(attempt.ID, true, "FLAG{x}", nil, 0)

	ranking := e.Ranking("agent-01")
	if ranking.AvgPercentile < 90 {
		t.Errorf("agent should have ~99%% percentile, got %.2f", ranking.AvgPercentile)
	}
}

// TestStats verifica las estadísticas.
func TestStats(t *testing.T) {
	e := NewEngine(DefaultConfig())

	// 3 retos: 2 resueltos, 1 no
	for i := 1; i <= 2; i++ {
		ch := helperChallenge("ch-"+string(rune('0'+i)), DifficultyEasy, CategoryWeb)
		_ = e.LoadChallenge(ch)
		attempt, _ := e.StartAttempt(ch.ID, "agent-01")
		_ = e.FinishAttempt(attempt.ID, true, "FLAG{x}", nil, 0)
	}
	ch3 := helperChallenge("ch-3", DifficultyMedium, CategoryCrypto)
	_ = e.LoadChallenge(ch3)

	stats := e.Stats()
	if stats.TotalChallenges != 3 {
		t.Errorf("expected 3 challenges, got %d", stats.TotalChallenges)
	}
	if stats.TotalSolved != 2 {
		t.Errorf("expected 2 solved, got %d", stats.TotalSolved)
	}
	if stats.ByCategory[CategoryWeb] != 2 {
		t.Errorf("expected 2 WEB, got %d", stats.ByCategory[CategoryWeb])
	}
	if stats.ByCategory[CategoryCrypto] != 1 {
		t.Errorf("expected 1 CRYPTO, got %d", stats.ByCategory[CategoryCrypto])
	}
}

// TestParallelismLimit verifica el límite de intentos paralelos.
func TestParallelismLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxParallelAttempts = 2

	e := NewEngine(cfg)
	ch := helperChallenge("ch-1", DifficultyEasy, CategoryWeb)
	_ = e.LoadChallenge(ch)

	// 2 intentos OK
	_, err := e.StartAttempt("ch-1", "agent-01")
	if err != nil {
		t.Fatalf("attempt 1 failed: %v", err)
	}
	_, err = e.StartAttempt("ch-1", "agent-02")
	if err != nil {
		t.Fatalf("attempt 2 failed: %v", err)
	}

	// 3ro debe fallar
	_, err = e.StartAttempt("ch-1", "agent-03")
	if err == nil {
		t.Error("3rd attempt should fail with parallelism limit")
	}
}

// TestUnsolvedList verifica el listado de no resueltos.
func TestUnsolvedList(t *testing.T) {
	e := NewEngine(DefaultConfig())

	_ = e.LoadChallenge(helperChallenge("ch-1", DifficultyEasy, CategoryWeb))
	_ = e.LoadChallenge(helperChallenge("ch-2", DifficultyEasy, CategoryWeb))
	_ = e.LoadChallenge(helperChallenge("ch-3", DifficultyEasy, CategoryWeb))

	// Resolver ch-1
	attempt, _ := e.StartAttempt("ch-1", "agent-01")
	_ = e.FinishAttempt(attempt.ID, true, "FLAG{x}", nil, 0)

	unsolved := e.ListUnsolvedChallenges()
	if len(unsolved) != 2 {
		t.Errorf("expected 2 unsolved, got %d", len(unsolved))
	}
}

// TestShutdown verifica el shutdown.
func TestShutdown(t *testing.T) {
	e := NewEngine(DefaultConfig())
	if err := e.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// LoadChallenge después de shutdown debe fallar
	err := e.LoadChallenge(helperChallenge("ch-1", DifficultyEasy, CategoryWeb))
	if err == nil {
		t.Error("LoadChallenge after shutdown should fail")
	}

	// StartAttempt después de shutdown debe fallar
	_, err = e.StartAttempt("ch-1", "agent-01")
	if err == nil {
		t.Error("StartAttempt after shutdown should fail")
	}
}

// TestListRankings verifica listado de rankings.
func TestListRankings(t *testing.T) {
	e := NewEngine(DefaultConfig())

	// Un reto por agente (no pueden compartir reto resuelto)
	agents := []string{"agent-01", "agent-02"}
	for i, agent := range agents {
		chID := "ch-" + string(rune('0'+i+1))
		ch := helperChallenge(chID, DifficultyEasy, CategoryWeb)
		_ = e.LoadChallenge(ch)

		attempt, err := e.StartAttempt(chID, agent)
		if err != nil {
			t.Fatalf("StartAttempt failed for %s: %v", agent, err)
		}
		_ = e.FinishAttempt(attempt.ID, true, "FLAG{x}", nil, 0)
	}

	rankings := e.ListRankings()
	if len(rankings) != 2 {
		t.Errorf("expected 2 rankings, got %d", len(rankings))
	}
}
