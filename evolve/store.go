package evolve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const stateFile = "state.json"
const maxStateBytes = 8 << 20

func readState(dir string) (*Snapshot, error) {
	if info, err := os.Lstat(dir); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return nil, errors.New("evolve: invalid state directory")
	}
	path := filepath.Join(dir, stateFile)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxStateBytes || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("evolve: state must be a bounded private regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxStateBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxStateBytes {
		return nil, errors.New("evolve: state is too large")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var state Snapshot
	if err := dec.Decode(&state); err != nil {
		return nil, fmt.Errorf("evolve: invalid stored state: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, errors.New("evolve: trailing state data")
	}
	if state.Version != 1 || state.Active == "" || len(state.Reports) > MaxHistory {
		return nil, errors.New("evolve: incompatible state")
	}
	for _, r := range state.Reports {
		if !validDigest(r.Dataset) || !validDigest(r.Contract) || r.Active == "" || len(r.Trials) > MaxCases*10 {
			return nil, errors.New("evolve: invalid stored report")
		}
	}
	return &state, nil
}

func writeState(dir string, state Snapshot) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("evolve: invalid state directory")
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(raw) > maxStateBytes {
		return errors.New("evolve: history size limit exceeded")
	}
	f, err := os.CreateTemp(dir, ".state-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, stateFile))
}
