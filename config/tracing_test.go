package config

import (
	"reflect"
	"testing"

	mqxconfig "github.com/gospacex/mqx/config"
)

func TestTracingConfig_FieldNamesMatchMQX(t *testing.T) {
	want := fieldNamesOf(reflect.TypeOf(mqxconfig.TracingConfig{}))
	got := fieldNamesOf(reflect.TypeOf(TracingConfig{}))
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("field names mismatch\nmqx: %v\nvectorx: %v", want, got)
	}
}

func fieldNamesOf(t reflect.Type) []string {
	out := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		out[i] = t.Field(i).Name
	}
	return out
}
