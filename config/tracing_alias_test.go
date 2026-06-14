package config

import (
	"reflect"
	"testing"

	mqxconfig "github.com/gospacex/mqx/config"
)

// TestTracingConfig_IsAliasToMQX ensures vectorx/config.TracingConfig
// and mqx/config.TracingConfig are the same type. If a future refactor
// unaliases the type, this test fails at compile time.
func TestTracingConfig_IsAliasToMQX(t *testing.T) {
	got := reflect.TypeOf(TracingConfig{})
	want := reflect.TypeOf(mqxconfig.TracingConfig{})
	if got != want {
		t.Fatalf("TracingConfig must be a type alias to mqxconfig.TracingConfig\n  got:  %v\n  want: %v", got, want)
	}
}
