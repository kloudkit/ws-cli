package mcp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

const (
	AnnotationAccess      = "mcp"
	AnnotationWhen        = "mcp.when"
	AnnotationFlags       = "mcp.flags"
	AnnotationDescription = "mcp.description"
)

type Access string

const (
	Read  Access = "read"
	Write Access = "write"
)

type Term struct {
	Group   string
	Prop    string
	Negated bool
}

func (t Term) String() string {
	term := t.Group + "." + t.Prop
	if t.Negated {
		return "!" + term
	}

	return term
}

type Spec struct {
	Access      Access
	When        []Term
	Flags       []string
	Description string
}

var (
	termPattern = regexp.MustCompile(`^(!?)([a-z][a-z0-9_]*)\.([a-z][a-z0-9_]*)$`)
	flagPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

func ParseSpec(annotations map[string]string) (Spec, bool, error) {
	access, annotated := annotations[AnnotationAccess]
	if !annotated {
		for _, key := range []string{AnnotationWhen, AnnotationFlags, AnnotationDescription} {
			if _, stray := annotations[key]; stray {
				return Spec{}, false, fmt.Errorf("annotation [%s] needs [%s]", key, AnnotationAccess)
			}
		}

		return Spec{}, false, nil
	}

	spec := Spec{Access: Access(access)}
	if spec.Access != Read && spec.Access != Write {
		return Spec{}, false, fmt.Errorf("annotation [%s] must be %q or %q, got %q", AnnotationAccess, Read, Write, access)
	}

	if when, ok := annotations[AnnotationWhen]; ok {
		for _, raw := range splitList(when) {
			match := termPattern.FindStringSubmatch(raw)
			if match == nil {
				return Spec{}, false, fmt.Errorf("annotation [%s] has a malformed term %q (want [!]<group>.<prop>)", AnnotationWhen, raw)
			}

			spec.When = append(spec.When, Term{Group: match[2], Prop: match[3], Negated: match[1] != ""})
		}
	}

	if flags, ok := annotations[AnnotationFlags]; ok {
		for _, name := range splitList(flags) {
			if !flagPattern.MatchString(name) {
				return Spec{}, false, fmt.Errorf("annotation [%s] has a malformed flag name %q", AnnotationFlags, name)
			}

			spec.Flags = append(spec.Flags, name)
		}
	}

	if description, ok := annotations[AnnotationDescription]; ok {
		if spec.Description = strings.TrimSpace(description); spec.Description == "" {
			return Spec{}, false, fmt.Errorf("annotation [%s] is empty", AnnotationDescription)
		}
	}

	return spec, true, nil
}

func ToolName(cmd *cobra.Command) string {
	return toolName(commandPath(cmd))
}

func toolName(path []string) string {
	return strings.ReplaceAll(strings.Join(path, "_"), "-", "_")
}

func commandPath(cmd *cobra.Command) []string {
	if !cmd.HasParent() {
		return nil
	}

	return append(commandPath(cmd.Parent()), cmd.Name())
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return parts
}
