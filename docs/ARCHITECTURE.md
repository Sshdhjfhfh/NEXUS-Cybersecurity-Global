# Arquitectura de Nexus IA v6.0

> **Documento de diseño técnico.** Define las interfaces propias que Nexus expone
> independientemente de ARTEX/norma.

---

## Principio rector

**Nexus no depende de nada que no controle al 100%.**

Todo lo externo se absorbe en `core/` con:
- Interfaces propias (Go interfaces).
- Implementaciones propias.
- Header con atribución AGPL.

---

## Paquetes del core

### `core/agent/` — Núcleo de agentes

```go
package agent

type Agent interface {
    ID() string
    Type() AgentType
    Execute(ctx context.Context, task Task) (Result, error)
    Telemetry() <-chan Telemetry
    Shutdown(ctx context.Context) error
}

type Planner interface {
    Agent
    Plan(ctx context.Context, goal Goal) ([]Intent, error)
}

type Worker interface {
    Agent
    Claim(ctx context.Context, intent Intent) error
}

type Coordinator interface {
    Agent
    Assign(intent Intent, worker Worker) error
    Collect() []Result
}

type Goal struct {
    ID          string
    Description string
    Scope       []string
    Priority    int
}

type Intent struct {
    ID         string
    GoalID     string
    Action     string
    AssetIDs   []string
    DependsOn  []string
    CreatedAt  time.Time
}

type Task struct {
    ID       string
    Payload  []byte
    Deadline time.Time
}

type Result struct {
    IntentID string
    Status   Status
    Findings []Finding
    Facts    []Fact
}

type Finding struct {
    ID         string
    Severity   Severity
    CVE        string
    CWE        string
    EvidenceID string
}

type Fact struct {
    Key   string
    Value string
    Node  string
}

type Telemetry struct {
    ComponentID string
    MetricName  string
    MetricValue float64
    Status      HealthStatus
    Timestamp   time.Time
}