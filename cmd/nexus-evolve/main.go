// nexus-evolve runs the same bounded retrieval evaluation used by Nexus agents.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/nexus-ia/nexus/core/memory"
	"github.com/nexus-ia/nexus/evolve"
	"github.com/nexus-ia/nexus/evolve/recall"
	"github.com/nexus-ia/nexus/lab/benchmark/recallfixture"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "No se pudo completar la evaluación:", err)
		os.Exit(1)
	}
}

func run() error {
	demo := flag.Bool("demo", false, "Evaluar el corpus sintético español del laboratorio")
	memoryDir := flag.String("memory-dir", "", "Directorio de memoria que se evaluará sin modificar sus notas")
	stateDir := flag.String("state-dir", "", "Directorio privado donde guardar las decisiones (opcional)")
	jsonOutput := flag.Bool("json", false, "Mostrar el informe completo como JSON")
	flag.Parse()
	if (*demo) == (*memoryDir != "") || flag.NArg() != 0 {
		return fmt.Errorf("indica -demo o -memory-dir, exclusivamente")
	}
	var mems []memory.Memory
	if *demo {
		mems = recallfixture.Memories()
	} else {
		info, err := os.Stat(*memoryDir)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("la ruta de memoria no es un directorio")
		}
		var errScan error
		mems, errScan = memory.NewStore(*memoryDir).Scan()
		if errScan != nil {
			return errScan
		}
	}
	idx, err := recall.NewIndex(mems)
	if err != nil {
		return err
	}
	e, err := evolve.New(evolve.Config{Dir: *stateDir, Baseline: recall.Legacy, Candidates: []string{recall.Terms, recall.Phrase}, Policy: evolve.DefaultPolicy()})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	r, err := e.Run(ctx, recall.NewBenchmark(idx))
	if err != nil {
		return err
	}
	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	decisions := map[string]string{"promoted": "mejora activada", "kept": "versión conservada", "rolled_back": "versión anterior recuperada", "insufficient": "evidencia insuficiente"}
	fmt.Printf("Evaluación de recuperación de memoria: %s\n", decisions[r.Decision])
	fmt.Printf("Estrategia anterior: %s\nEstrategia activa: %s\n", r.Before, r.Active)
	if r.Candidate != "" {
		fmt.Printf("Casos reservados: %d; aciertos anteriores: %d; aciertos candidatos: %d\n", r.Holdout.Cases, r.Holdout.BaselinePassed, r.Holdout.CandidatePassed)
	}
	fmt.Println("La medición evalúa búsquedas por título; no demuestra mejoras generales del modelo.")
	return nil
}
