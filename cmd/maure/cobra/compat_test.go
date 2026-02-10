package maurecobra

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNormalizeLegacyArgsParseLogFormat(t *testing.T) {
	raw := []string{"--index", "/tmp/i", "parse-log", "app.log", "--format=json"}
	var stderr bytes.Buffer
	got := NormalizeLegacyArgs(raw, &stderr)
	want := []string{"--index", "/tmp/i", "parse-log", "app.log", "--log-format=json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized args mismatch\nwant=%v\ngot=%v", want, got)
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected deprecation warning")
	}
}

func TestHasLegacyFlagOrder(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "legacy", args: []string{"search", "error", "--group=level"}, want: true},
		{name: "recommended", args: []string{"search", "--group=level", "error"}, want: false},
		{name: "no flags", args: []string{"search", "error"}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasLegacyFlagOrder(tc.args, "search"); got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}
