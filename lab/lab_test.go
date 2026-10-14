package lab

import (
	"context"
	"errors"
	"github.com/nexus-ia/nexus/lab/network"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
	"testing"
)

func TestGateRequiresEveryControlAndHonorsStop(t *testing.T) {
	scope, _ := network.NewScope("172.30.4.0/24")
	if _, err := NewGate(scope, Checks{}); err == nil {
		t.Fatal("missing checks accepted")
	}
	calls := []string{}
	check := func(name string) Check {
		return func(context.Context, Action) error { calls = append(calls, name); return nil }
	}
	checks := Checks{check("identity"), check("opsec"), check("ethics"), check("monitor")}
	gate, _ := NewGate(scope, checks)
	action := Action{"blue-1", "task-1", "analyze", "172.30.4.2"}
	if err := gate.Authorize(context.Background(), action); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"identity", "monitor", "opsec", "ethics"}) {
		t.Fatal(calls)
	}
	calls = nil
	action.Target = "8.8.8.8"
	if err := gate.Authorize(context.Background(), action); err == nil || len(calls) != 0 {
		t.Fatal("out-of-scope action reached controls")
	}
	action.Target = "172.30.4.2"
	checks.Ethics = func(context.Context, Action) error { return errors.New("STOP") }
	gate, _ = NewGate(scope, checks)
	if err := gate.Authorize(context.Background(), action); err == nil || !strings.Contains(err.Error(), "STOP") {
		t.Fatal("ethical STOP was lost")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = nil
	if err := gate.Authorize(ctx, action); !errors.Is(err, context.Canceled) || len(calls) != 0 {
		t.Fatal("cancelled action was authorized")
	}
	var empty *Gate
	if err := empty.Authorize(context.Background(), action); err == nil {
		t.Fatal("nil gate allowed action")
	}
}
func TestAllScenarioProfilesAreIsolatedAndPinned(t *testing.T) {
	for _, profile := range Profiles() {
		t.Run(string(profile.Kind), func(t *testing.T) {
			images := map[string]string{}
			for _, role := range profile.Roles {
				images[role] = "local/" + role + "@sha256:" + strings.Repeat("a", 64)
			}
			scenario := Scenario{profile.Kind, images}
			data, err := scenario.Compose()
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err = yaml.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			network := document["networks"].(map[string]any)["lab"].(map[string]any)
			if network["internal"] != true {
				t.Fatal("lab network has external access")
			}
			for _, service := range document["services"].(map[string]any) {
				s := service.(map[string]any)
				for _, forbidden := range []string{"ports", "privileged", "network_mode", "volumes", "devices"} {
					if _, ok := s[forbidden]; ok {
						t.Fatalf("unsafe service option %s", forbidden)
					}
				}
				if s["pull_policy"] != "never" {
					t.Fatal("implicit image download allowed")
				}
			}
			images[profile.Roles[0]] = "latest"
			if _, err = scenario.Compose(); err == nil {
				t.Fatal("unpinned image accepted")
			}
		})
	}
	if _, err := (Scenario{Kind: "unknown"}).Compose(); err == nil {
		t.Fatal("unknown scenario accepted")
	}
}
