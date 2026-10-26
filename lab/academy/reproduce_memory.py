#!/usr/bin/env python3
"""Reproduce historical memory-boundary regressions with temporary synthetic files.

Copies one source file from HEAD and the working tree into isolated temporary
packages. Does not alter the repository history, deployed services or user data.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

from run import write_json

ROOT = Path(__file__).resolve().parents[2]
SOURCE = "core/memory/memory.go"
PROBE = r'''package memory
import ("os"; "path/filepath"; "strings"; "testing")
func must(t *testing.T, err error) { t.Helper(); if err != nil { t.Fatal(err) } }
func TestBoundaryFile(t *testing.T) {
    dir := t.TempDir()
    other := filepath.Join(t.TempDir(), "outside.md")
    must(t, os.WriteFile(other, []byte("---\nname: outside\n---\nSYNTHETIC_OUTSIDE"), 0600))
    must(t, os.Symlink(other, filepath.Join(dir, "linked.md")))
    note, err := NewStore(dir).Read("linked")
    if err == nil && strings.Contains(note.Content, "SYNTHETIC_OUTSIDE") {
        t.Fatal("BOUNDARY_FILE_LEAK")
    }
}
func TestBoundaryDirectory(t *testing.T) {
    other := NewStore(t.TempDir())
    must(t, other.Save(Memory{Name:"outside", Content:"SYNTHETIC_OUTSIDE"}))
    linked := filepath.Join(t.TempDir(), "memory")
    must(t, os.Symlink(other.Dir, linked))
    note, err := NewStore(linked).Read("outside")
    if err == nil && strings.Contains(note.Content, "SYNTHETIC_OUTSIDE") {
        t.Fatal("BOUNDARY_DIRECTORY_LEAK")
    }
}
func TestBoundaryIndex(t *testing.T) {
    dir := t.TempDir()
    other := filepath.Join(t.TempDir(), "index.md")
    must(t, os.WriteFile(other, []byte("SYNTHETIC_OUTSIDE"), 0600))
    must(t, os.Symlink(other, filepath.Join(dir, "MEMORY.md")))
    if strings.Contains(NewStore(dir).Index(), "SYNTHETIC_OUTSIDE") {
        t.Fatal("BOUNDARY_INDEX_LEAK")
    }
}
'''
TESTS = {"TestBoundaryFile", "TestBoundaryDirectory", "TestBoundaryIndex"}


def reproduce(out):
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    historical = subprocess.check_output(["git", "show", f"{commit}:{SOURCE}"], cwd=ROOT)
    current = (ROOT / SOURCE).read_bytes()
    results = {}
    for name, code in (("historical", historical), ("working_tree", current)):
        with tempfile.TemporaryDirectory(prefix="nexus-memory-probe-") as directory:
            work = Path(directory)
            (work / "memory.go").write_bytes(code)
            (work / "boundary_test.go").write_text(PROBE)
            for filename in ("go.mod", "go.sum"):
                (work / filename).write_bytes((ROOT / filename).read_bytes())
            env = dict(os.environ, GOCACHE="/tmp/nexus-academy-go-cache", GOPROXY="off", GOSUMDB="off", GOWORK="off")
            result = subprocess.run(["go", "test", "-mod=readonly", "-json", "-count=1", "."],
                                    cwd=work, env=env, text=True, capture_output=True, timeout=180)
            events = []
            for line in result.stdout.splitlines():
                try:
                    events.append(json.loads(line))
                except ValueError:
                    pass
            outcomes = {e["Test"]: e["Action"] for e in events if e.get("Test") in TESTS
                        and e.get("Action") in {"pass", "fail", "skip"}}
            results[name] = {"source_sha256": hashlib.sha256(code).hexdigest(),
                             "returncode": result.returncode, "tests": outcomes,
                             "stdout": result.stdout, "stderr": result.stderr}
            print(json.dumps({"version": name, "tests": outcomes, "returncode": result.returncode}), flush=True)
    old = results["historical"]
    new = results["working_tree"]
    expected_markers = ["BOUNDARY_FILE_LEAK", "BOUNDARY_DIRECTORY_LEAK", "BOUNDARY_INDEX_LEAK"]
    confirmed = (old["returncode"] == 1 and old["tests"] == {name: "fail" for name in TESTS}
                 and all(marker in old["stdout"] for marker in expected_markers)
                 and new["returncode"] == 0 and new["tests"] == {name: "pass" for name in TESTS})
    report = {"commit": commit, "source": SOURCE, "reproduced_and_corrected": confirmed,
              "results": results, "probe_sha256": hashlib.sha256(PROBE.encode()).hexdigest(),
              "limitation": "Local attacker must be able to create the tested symlinks. No remote exploit, novelty claim, ancestor-path or race guarantee."}
    write_json(out / "memory-regressions.json", report)
    if not confirmed:
        raise RuntimeError("Historical/current boundary evidence did not match; inspect report")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True, type=Path)
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=False, mode=0o700)
    reproduce(args.out)
