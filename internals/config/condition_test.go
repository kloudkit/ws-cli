package config

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

const conditionYAML = `envs:
  features:
    properties:
      store_url:
        type: string
        default: null
      offline:
        type: boolean
        default: false
      token:
        type: string
        default: null
        secret: true
`

func TestParseCondition(t *testing.T) {
	for term, want := range map[string]Condition{
		"features.store_url":   {Group: "features", Prop: "store_url"},
		"!features.store_url":  {Group: "features", Prop: "store_url", Negated: true},
		"ai.mcp_allow_write":   {Group: "ai", Prop: "mcp_allow_write"},
		"!group2.prop_3_value": {Group: "group2", Prop: "prop_3_value", Negated: true},
	} {
		condition, err := ParseCondition(term)
		assert.NilError(t, err, term)
		assert.Equal(t, condition, want)
		assert.Equal(t, condition.String(), term)
	}

	for _, term := range []string{"", "features", "features.", ".store_url", "Features.store_url", "features.store_url.x", "!", "!!features.store_url", "features store_url", " features.store_url", "WS_FEATURES_STORE_URL"} {
		_, err := ParseCondition(term)
		assert.ErrorContains(t, err, "malformed term", "%q", term)
		assert.ErrorContains(t, err, "[!]<group>.<prop>", "%q", term)
	}
}

func TestHolds(t *testing.T) {
	cases := map[string]struct {
		env   map[string]string
		terms []string
		want  bool
	}{
		"StringUnset":             {nil, []string{"features.store_url"}, false},
		"StringUnsetNegated":      {nil, []string{"!features.store_url"}, true},
		"StringSet":               {map[string]string{"WS_FEATURES_STORE_URL": "https://store"}, []string{"features.store_url"}, true},
		"StringSetNegated":        {map[string]string{"WS_FEATURES_STORE_URL": "https://store"}, []string{"!features.store_url"}, false},
		"BooleanDefaultFalse":     {nil, []string{"features.offline"}, false},
		"BooleanDefaultNegated":   {nil, []string{"!features.offline"}, true},
		"BooleanTrue":             {map[string]string{"WS_FEATURES_OFFLINE": "yes"}, []string{"features.offline"}, true},
		"BooleanFalseIsNotEmpty":  {map[string]string{"WS_FEATURES_OFFLINE": "false"}, []string{"features.offline"}, false},
		"BooleanUnparsable":       {map[string]string{"WS_FEATURES_OFFLINE": "maybe"}, []string{"features.offline"}, false},
		"BooleanUnparsableNegate": {map[string]string{"WS_FEATURES_OFFLINE": "maybe"}, []string{"!features.offline"}, false},
		"Undeclared":              {map[string]string{"WS_FEATURES_GHOST": "1"}, []string{"features.ghost"}, false},
		"UndeclaredNegated":       {nil, []string{"!features.ghost"}, false},
		"SecretSet":               {map[string]string{"WS_FEATURES_TOKEN": "hunter2"}, []string{"features.token"}, true},
		"AllMustHold":             {map[string]string{"WS_FEATURES_STORE_URL": "https://store"}, []string{"features.store_url", "features.offline"}, false},
		"AllHold":                 {map[string]string{"WS_FEATURES_STORE_URL": "https://store"}, []string{"features.store_url", "!features.offline"}, true},
		"NoConditions":            {nil, nil, true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_installFixture(t, conditionYAML)

			for _, key := range []string{"WS_FEATURES_STORE_URL", "WS_FEATURES_OFFLINE", "WS_FEATURES_TOKEN"} {
				t.Setenv(key, "")
			}

			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			var conditions []Condition
			for _, term := range tc.terms {
				condition, err := ParseCondition(term)
				assert.NilError(t, err)

				conditions = append(conditions, condition)
			}

			holds, err := Holds(conditions)
			assert.NilError(t, err)
			assert.Equal(t, holds, tc.want)
		})
	}

	t.Run("UnreadableReference", func(t *testing.T) {
		t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "missing.yaml"))

		_, err := Holds([]Condition{{Group: "features", Prop: "store_url"}})
		assert.ErrorContains(t, err, "cannot read")
	})
}
