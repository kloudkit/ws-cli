package config

import (
	"fmt"
	"regexp"
)

type Condition struct {
	Group   string
	Prop    string
	Negated bool
}

var conditionPattern = regexp.MustCompile(`^(!?)([a-z][a-z0-9_]*)\.([a-z][a-z0-9_]*)$`)

func ParseCondition(term string) (Condition, error) {
	match := conditionPattern.FindStringSubmatch(term)
	if match == nil {
		return Condition{}, fmt.Errorf("malformed term %q (want [!]<group>.<prop>)", term)
	}

	return Condition{Group: match[2], Prop: match[3], Negated: match[1] != ""}, nil
}

func (c Condition) String() string {
	term := c.Group + "." + c.Prop
	if c.Negated {
		return "!" + term
	}

	return term
}

func Holds(conditions []Condition) (bool, error) {
	for _, condition := range conditions {
		key := RuntimeKey(condition.Group, condition.Prop)

		prop, declared, err := LookupProperty(key)
		if err != nil {
			return false, err
		}

		if !declared {
			return false, nil
		}

		value, err := ResolveKey(key)
		if err != nil {
			return false, nil
		}

		truth := value != ""
		if truth && prop.Type == "boolean" {
			if truth, err = ParseBool(value); err != nil {
				return false, nil
			}
		}

		if truth == condition.Negated {
			return false, nil
		}
	}

	return true, nil
}
