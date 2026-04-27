package features

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

func RunPlaybook(featurePath string, vars map[string]any) error {
	args := []string{featurePath}

	if len(vars) > 0 {
		keys := make([]string, 0, len(vars))
		for key := range vars {
			keys = append(keys, key)
		}
		slices.Sort(keys)

		extraVars := make([]string, 0, len(keys))
		for _, key := range keys {
			extraVars = append(extraVars, fmt.Sprintf("%s=%v", key, vars[key]))
		}
		args = append(args, "--extra-vars", strings.Join(extraVars, " "))
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Env = append(os.Environ(),
		"ANSIBLE_DISPLAY_OK_HOSTS=0",
		"ANSIBLE_DISPLAY_FAILED_STDERR=0",
		"ANSIBLE_DISPLAY_SKIPPED_HOSTS=0",
		"ANSIBLE_SHOW_CUSTOM_STATS=0",
		"ANSIBLE_STDOUT_CALLBACK=community.general.unixy",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
