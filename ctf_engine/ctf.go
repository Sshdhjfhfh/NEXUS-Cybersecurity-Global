// Nexus IA v6.0 — CTF Engine (Capture The Flag autónomo)
//
// El swarm compite en CTFs autónomamente:
//   - Descarga retos de HTB (offline), PicoCTF, CTFtime
//   - El swarm ataca retos en paralelo sin intervención humana
//   - Mide: tiempo de resolución, técnicas usadas, pistas necesarias
//   - Genera ranking: ¿en qué percentil humano está el agente?
//   - Métrica clave: ¿mejora el ranking del agente con el tiempo?
//
// Basado en CALBench (2026): agentes IA compiten vs humanos en CTF real.
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package ctf_engine

import (
	"time"
)

// Platform identifica la plataforma CTF de origen.
type Platform string

const (
	PlatformHTB      Platform = "HTB"       // Hack The Box
	PlatformPicoCTF  Platform = "PICOCFTF"  // PicoCTF
	PlatformCTFtime  Platform = "CTFTIME"   // CTFtime
	PlatformLocal    Platform = "LOCAL"     // Local test
	PlatformOther    Platform = "OTHER"
)

// Category es la categoría del reto CTF.
type Category string

const (
	CategoryWeb       Category = "WEB"
	CategoryPwn       Category = "PWN"
	CategoryCrypto    Category = "CRYPTO"
	CategoryReverse   Category = "REVERSE"
	CategoryForensics Category = "FORENSICS"
	CategoryMisc      Category = "MISC"
	CategoryBlockchain Category = "BLOCKCHAIN"
	CategoryHardware  Category = "HARDWARE"
)

// Difficulty clasifica la dificultad del reto.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "EASY"
	DifficultyMedium Difficulty = "MEDIUM"
	DifficultyHard   Difficulty = "HARD"
	DifficultyInsane Difficulty = "INSANE"
)

// Challenge es un reto CTF.
type Challenge struct {
	// Identificación
	ID          string      // ID único del reto
	Name        string      // Nombre del reto
	Platform    Platform    // Plataforma de origen
	Category    Category    // Categoría
	Difficulty  Difficulty  // Dificultad

	// Contenido
	Description string      // Descripción del reto
	Files       []string    // Archivos adjuntos (rutas)
	Hints       []Hint      // Pistas disponibles
	FlagFormat  string      // Formato esperado de la flag (ej: HTB{...})
	Points      int         // Puntos que da

	// Estado
	Solved      bool        // ¿Resuelto?
	SolvedBy    string      // AgentID que lo resolvió
	SolvedAt    *time.Time  // Cuándo se resolvió
	Flag        string      // Flag encontrada

	// Metadatos
	Metadata    map[string]string
	CreatedAt   time.Time
}

// Hint es una pista del reto.
type Hint struct {
	Index       int       // Índice (1, 2, 3...)
	Text        string    // Texto de la pista
	PointsCost  int       // Costo en puntos
	Used        bool      // ¿Ya usada por el agente?
	UsedBy      string    // AgentID que la usó
	UsedAt      *time.Time
}

// Attempt es un intento de resolución de un reto.
type Attempt struct {
	// Identificación
	ID          string    // ID único del intento
	ChallengeID string    // Reto objetivo
	AgentID     string    // Agente que intenta

	// Timing
	StartedAt   time.Time
	FinishedAt  *time.Time

	// Resultado
	Solved      bool      // ¿Resolvió?
	Flag        string    // Flag si resolvió
	TimeSpent   time.Duration
	HintsUsed   int       // Cuántas pistas necesitó

	// Técnicas usadas
	Techniques  []string  // Técnicas aplicadas
	MITRE_TTPs  []string  // MITRE ATT&CK IDs

	// Metadatos
	TraceID     string
	SessionID   string
	Notes       string
}

// SolveMetrics son las métricas de resolución de un reto.
type SolveMetrics struct {
	ChallengeID  string
	Platform     Platform
	Category     Category
	Difficulty   Difficulty

	// Métricas propias
	TimeSpent    time.Duration
	HintsUsed    int
	Techniques   []string

	// Comparación con humanos
	HumanAvgTime time.Duration
	HumanAvgHints float64
	Percentile   float64 // 0-100. Ej: 75 = mejor que 75% de humanos
}

// AgentRanking es el ranking de un agente.
type AgentRanking struct {
	AgentID         string
	TotalSolves     int
	TotalAttempts   int
	SolveRate       float64       // solves / attempts
	AvgTimeSpent    time.Duration
	AvgHintsUsed    float64
	AvgPercentile   float64       // promedio de percentiles vs humanos

	// Por dificultad
	EasySolves      int
	MediumSolves    int
	HardSolves      int
	InsaneSolves    int

	// Por categoría
	ByCategory      map[Category]int

	// Tendencia temporal
	FirstSolveAt    *time.Time
	LastSolveAt     *time.Time
	WeeklyProgress  []WeeklyProgress
}

// WeeklyProgress es el progreso semanal de un agente.
type WeeklyProgress struct {
	WeekStart    time.Time
	Solves       int
	AvgTime      time.Duration
	AvgPercentile float64
}

// Config configura el CTF Engine.
type Config struct {
	// ¿Modo offline? (obligatorio por Zero Trust — no contactar HTB real)
	OfflineMode bool

	// Plataformas activas.
	EnabledPlatforms []Platform

	// ¿Cargar retos automáticamente al arrancar?
	AutoLoad bool

	// Máximo de intentos paralelos.
	MaxParallelAttempts int

	// Timeout por intento.
	AttemptTimeout time.Duration

	// Número máximo de pistas a usar por reto.
	MaxHintsPerChallenge int

	// Directorio donde se guardan retos y flags.
	DataDir string
}

// DefaultConfig devuelve la configuración segura por defecto.
func DefaultConfig() Config {
	return Config{
		OfflineMode:          true, // SIEMPRE true por Zero Trust
		EnabledPlatforms:     []Platform{PlatformLocal, PlatformPicoCTF},
		AutoLoad:             false,
		MaxParallelAttempts:  4,
		AttemptTimeout:       30 * time.Minute,
		MaxHintsPerChallenge: 2,
		DataDir:              "data/ctf",
	}
}

// Engine es la interfaz del CTF Engine.
type Engine interface {
	// LoadChallenge carga un reto al engine.
	LoadChallenge(ch *Challenge) error

	// ListChallenges devuelve los retos disponibles.
	ListChallenges() []*Challenge

	// GetChallenge recupera un reto por ID.
	GetChallenge(id string) (*Challenge, error)

	// StartAttempt inicia un intento de resolución.
	StartAttempt(challengeID, agentID string) (*Attempt, error)

	// FinishAttempt cierra un intento.
	FinishAttempt(attemptID string, solved bool, flag string, techniques []string, hintsUsed int) error

	// ListAttempts devuelve los intentos de un reto.
	ListAttempts(challengeID string) []*Attempt

	// Metrics devuelve las métricas de un reto resuelto.
	Metrics(challengeID string) *SolveMetrics

	// Ranking devuelve el ranking de un agente.
	Ranking(agentID string) *AgentRanking

	// Stats devuelve estadísticas globales.
	Stats() EngineStats
}

// EngineStats lleva estadísticas del engine.
type EngineStats struct {
	TotalChallenges   int
	TotalSolved       int
	TotalAttempts     int
	TotalAgents       int
	ByPlatform        map[Platform]int
	ByCategory        map[Category]int
	ByDifficulty      map[Difficulty]int
}
