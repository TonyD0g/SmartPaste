package secret

import "testing"

func TestEnvironmentKeyOverride(t *testing.T) {
	t.Setenv("SMARTPASTE_DIRECT_JEV_KEY", "  env-key  ")
	store := New()

	value, err := store.Get()
	if err != nil {
		t.Fatal(err)
	}
	if value != "env-key" {
		t.Fatalf("unexpected key: %q", value)
	}
	if !store.FromEnvironment() {
		t.Fatal("expected environment override to be reported")
	}
}
