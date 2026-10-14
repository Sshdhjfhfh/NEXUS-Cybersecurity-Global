package prediction

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestCalibrationReadinessAndAtomicFeedback(t *testing.T) {
	e, _ := New("auth", "headers")
	if e.Rank()[0].Ready || e.Rank()[0].Probability != 0.5 {
		t.Fatal("prior is overconfident")
	}
	for i := 0; i < 50; i++ {
		if err := e.Record(Outcome{fmt.Sprint(i), "auth", i < 40}); err != nil {
			t.Fatal(err)
		}
	}
	top := e.Rank()[0]
	if top.Class != "auth" || !top.Ready || top.Samples != 50 || math.Abs(top.Probability-41.0/52) > 1e-9 || top.Lower > 0.8 || top.Upper < 0.8 {
		t.Fatal(top)
	}
	before := e.Rank()
	if err := e.Record(Outcome{"new", "headers", true}, Outcome{"0", "auth", true}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if !reflect.DeepEqual(before, e.Rank()) {
		t.Fatal("invalid batch partially committed")
	}
	if err := e.Record(Outcome{"x", "unknown", true}); err == nil {
		t.Fatal("unknown class accepted")
	}
}
func TestConcurrentFeedbackAndPersistence(t *testing.T) {
	e, _ := New("class")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := e.Record(Outcome{fmt.Sprint(i), "class", i%2 == 0}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	path := filepath.Join(t.TempDir(), "prediction.json")
	if err := e.Save(path); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(e.Rank(), restored.Rank()) {
		t.Fatal("snapshot changed predictions")
	}
	if err := restored.Record(Outcome{"1", "class", true}); err == nil {
		t.Fatal("restoring lost duplicate protection")
	}
	data, _ := os.ReadFile(path)
	var state snapshot
	json.Unmarshal(data, &state)
	state.Counts["class"] = Counts{Success: 100}
	data, _ = json.Marshal(state)
	os.WriteFile(path, data, 0600)
	if _, err := Load(path); err == nil {
		t.Fatal("corrupt counts accepted")
	}
	os.WriteFile(path, append(data, []byte(` {}`)...), 0600)
	if _, err := Load(path); err == nil {
		t.Fatal("trailing JSON accepted")
	}
}
