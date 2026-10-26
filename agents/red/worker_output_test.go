package agent

import (
	"errors"
	"testing"

	"github.com/nexus-ia/nexus/db"
)

func TestLatestWorkerOutputBeyondFirstPage(t *testing.T) {
	rows := make([]db.Activity, 2105)
	for i := range rows {
		rows[i] = db.Activity{ID: int64(i + 1), Kind: "tool_result"}
	}
	rows[0].Kind = "text"
	rows[999].Kind = "result"
	rows[2102] = db.Activity{ID: 2103, Kind: "result", Summary: "final verified conclusion"}
	rows[2104].Kind = "text"
	var calls int
	got, err := latestWorkerOutput(func(since int64, limit int) ([]db.Activity, int64, error) {
		calls++
		end := min(int(since)+limit, len(rows))
		return rows[int(since):end], int64(end), nil
	})
	if err != nil || got == nil || got.ID != 2103 || calls != 3 {
		t.Fatalf("got=%+v calls=%d err=%v", got, calls, err)
	}
}

func TestLatestWorkerOutputUsesLastTextWithoutResult(t *testing.T) {
	got, err := latestWorkerOutput(func(int64, int) ([]db.Activity, int64, error) {
		return []db.Activity{{ID: 1, Kind: "text", Summary: "initial"}, {ID: 2, Kind: "text", Summary: "latest"}}, 2, nil
	})
	if err != nil || got == nil || got.Summary != "latest" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestLatestWorkerOutputDoesNotHideReadErrors(t *testing.T) {
	want := errors.New("read failed")
	_, err := latestWorkerOutput(func(int64, int) ([]db.Activity, int64, error) { return nil, 0, want })
	if !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
	_, err = latestWorkerOutput(func(int64, int) ([]db.Activity, int64, error) { return make([]db.Activity, 1000), 0, nil })
	if err == nil {
		t.Fatal("non-advancing page would loop forever")
	}
	got, err := latestWorkerOutput(func(int64, int) ([]db.Activity, int64, error) { return nil, 0, nil })
	if err != nil || got != nil {
		t.Fatalf("empty got=%+v err=%v", got, err)
	}
}
