package template

import (
	"encoding/json"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestAll_ReturnsEmptySliceNotNil ensures that when no custom templates are
// configured and no builtin .handlebars files remain (PR #46 deleted them),
// All() returns an empty slice rather than a nil slice. Encoding a nil slice
// as JSON produces `null`, which breaks frontend code that expects an array
// (see issue #71 — `templates[0]` crashes with TypeError).
func TestAll_ReturnsEmptySliceNotNil(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{ImportTemplates: []config.ImportTemplate{}})
	t.Cleanup(func() { config.SetConfigForTest(prev) })

	templates := All()
	assert.NotNil(t, templates, "All() must not return nil even with zero templates")
	assert.Equal(t, 0, len(templates), "All() should return an empty slice when nothing is configured")

	// JSON-serialisation contract: the frontend calls JSON.parse and indexes
	// into `templates[0]`. The wire shape must be `[]`, not `null`.
	payload, err := json.Marshal(map[string]any{"templates": templates})
	assert.NoError(t, err)
	assert.Equal(t, `{"templates":[]}`, string(payload),
		"templates field must serialise to [] (got %s)", string(payload))
}

// TestAll_IncludesCustomTemplates verifies the happy path: custom templates
// configured in paisa.yaml are still returned. Guards against an over-eager
// rewrite that drops them while fixing the nil-slice bug.
func TestAll_IncludesCustomTemplates(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{
		ImportTemplates: []config.ImportTemplate{
			{Name: "my-bank", Content: "{{date}} {{payee}}"},
			{Name: "broker", Content: "{{date}} buy {{symbol}}"},
		},
	})
	t.Cleanup(func() { config.SetConfigForTest(prev) })

	templates := All()
	assert.GreaterOrEqual(t, len(templates), 2)

	names := map[string]string{}
	for _, tpl := range templates {
		if tpl.TemplateType == Custom {
			names[tpl.Name] = tpl.Content
		}
	}
	assert.Equal(t, "{{date}} {{payee}}", names["my-bank"])
	assert.Equal(t, "{{date}} buy {{symbol}}", names["broker"])
}
