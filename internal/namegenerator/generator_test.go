package namegenerator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidGenerationName(t *testing.T) {
	tests := []struct {
		name  string
		sufix bool
	}{
		{"contains number", false},
		{"not contains number", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := GenerateName(tt.sufix)

			assert.Contains(t, name, "_")
			if tt.sufix {
				assert.Regexp(t, `\d`, name)
			} else {
				assert.NotRegexp(t, `\d`, name)
			}
		})
	}
}

func BenchmarkGenerateName(b *testing.B) {
	b.ReportAllocs()

	var name string
	for b.Loop() {
		name = GenerateName(true)
	}
	b.Log("Result:", name)
}
