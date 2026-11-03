package features

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

const _bad = "invalid yaml [[["

func _play(desc string) string {
	return "---\n- name: " + desc + "\n  gather_facts: false\n  hosts: workspace\n"
}

func _write(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		assert.NilError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
}

func TestParseFeatureFile(t *testing.T) {
	cases := []struct {
		name, content, desc string
		vars                []string
		err                 string
	}{
		{"WithVars", _play("Install Test Feature") + "  vars:\n    repo: x\n    gpg: y\n  tasks:\n    - name: t\n", "Install Test Feature", []string{"gpg", "repo"}, ""},
		{"NoVars", _play("Install Feature Without Vars"), "Install Feature Without Vars", nil, ""},
		{"StripsJinja", _play(`"Install .NET {{ dotnet_version }}"`), "Install .NET", nil, ""},
		{"StripsInnerJinja", _play(`"Install {{ x }} tools"`), "Install tools", nil, ""},
		{"EmptyArray", "---\n[]\n", "", nil, "no playbook tasks found"},
		{"Invalid", _bad, "", nil, "failed to parse YAML"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			_write(t, dir, map[string]string{"my-feature.yaml": c.content})

			feature, err := ParseFeatureFile(filepath.Join(dir, "my-feature.yaml"))
			if c.err != "" {
				assert.ErrorContains(t, err, c.err)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, &Feature{Name: "my-feature", Description: c.desc, Vars: c.vars}, feature)
		})
	}
}

func TestListFeatures(t *testing.T) {
	cases := []struct {
		name         string
		system, user map[string]string
		noUserDir    bool
		noSystemDir  bool
		want         map[string]FeatureSource
		descs        map[string]string
		warnings     int
	}{
		{name: "SingleDirSkipsNonYAML", system: map[string]string{"a.yaml": _play("A"), "b.yaml": _play("B"), "c.txt": "x"}, noUserDir: true,
			want: map[string]FeatureSource{"a": SourceSystem, "b": SourceSystem}},
		{name: "MalformedWarns", system: map[string]string{"good.yaml": _play("Good"), "bad.yaml": _bad}, user: map[string]string{},
			want: map[string]FeatureSource{"good": SourceSystem}, warnings: 1},
		{name: "UserOverridesSystem", system: map[string]string{"git.yaml": _play("System Git")}, user: map[string]string{"git.yaml": _play("User Git")},
			want: map[string]FeatureSource{"git": SourceOverride}, descs: map[string]string{"git": "User Git"}},
		{name: "MarksSource", system: map[string]string{"sys.yaml": _play("S"), "shared.yaml": _play("SS")}, user: map[string]string{"shared.yaml": _play("US"), "usr.yaml": _play("U")},
			want: map[string]FeatureSource{"sys": SourceSystem, "shared": SourceOverride, "usr": SourceUser}},
		{name: "UserDirAbsent", system: map[string]string{"git.yaml": _play("G")}, noUserDir: true,
			want: map[string]FeatureSource{"git": SourceSystem}},
		{name: "SystemDirAbsent", user: map[string]string{"git.yaml": _play("G")}, noSystemDir: true,
			want: map[string]FeatureSource{"git": SourceUser}},
		{name: "MalformedUserNoTwin", system: map[string]string{"good.yaml": _play("G")}, user: map[string]string{"bad.yaml": _bad},
			want: map[string]FeatureSource{"good": SourceSystem}, warnings: 1},
		{name: "MalformedUserShadowsSystemNoFallback", system: map[string]string{"git.yaml": _play("G")}, user: map[string]string{"git.yaml": _bad},
			want: map[string]FeatureSource{}, warnings: 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			system, user := t.TempDir(), t.TempDir()
			_write(t, system, c.system)
			_write(t, user, c.user)
			if c.noSystemDir {
				system = filepath.Join(system, "missing")
			}
			if c.noUserDir {
				user = filepath.Join(user, "missing")
			}

			result, err := ListFeatures([]string{system, user})
			assert.NilError(t, err)
			assert.Equal(t, c.warnings, len(result.Warnings))

			got := map[string]FeatureSource{}
			for _, f := range result.Features {
				got[f.Name] = f.Source
				if want, ok := c.descs[f.Name]; ok {
					assert.Equal(t, want, f.Description)
				}
			}
			assert.DeepEqual(t, c.want, got)
		})
	}
}

func TestInfoFeatureAndResolvePath(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_write(t, system, map[string]string{"git.yaml": _play("System Git"), "sys.yaml": _play("System Only"), "broken.yaml": _play("System Broken")})
	_write(t, user, map[string]string{"git.yaml": _play("User Git"), "broken.yaml": _bad})
	dirs := []string{system, user}

	feature, err := InfoFeature(dirs, "git")
	assert.NilError(t, err)
	assert.DeepEqual(t, &Feature{Name: "git", Description: "User Git", Source: SourceOverride}, feature)

	feature, err = InfoFeature(dirs, "sys")
	assert.NilError(t, err)
	assert.Equal(t, SourceSystem, feature.Source)

	_, err = InfoFeature(dirs, "broken")
	assert.ErrorContains(t, err, "failed to parse YAML")

	_, err = InfoFeature(dirs, "nope")
	assert.Error(t, err, "feature 'nope' not found")

	path, err := ResolveFeaturePath(dirs, "git")
	assert.NilError(t, err)
	assert.Equal(t, filepath.Join(user, "git.yaml"), path)

	_, err = ResolveFeaturePath(dirs, "nope")
	assert.Error(t, err, "feature 'nope' not found")
}
