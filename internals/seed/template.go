package seed

import (
	"fmt"
	"regexp"
	"strings"
)

const secretsPrefix = "secrets."

var tokenRe = regexp.MustCompile(`\$\{([^}]*)\}`)

func referencesSecrets(content []byte) bool {
	return strings.Contains(string(content), "${"+secretsPrefix)
}

func renderTemplate(content []byte, vars Vars, secret func(string) ([]byte, error)) ([]byte, error) {
	var failure error

	rendered := tokenRe.ReplaceAllFunc(content, func(match []byte) []byte {
		if failure != nil {
			return nil
		}

		token := string(match[2 : len(match)-1])

		switch token {
		case "ws_home":
			return []byte(vars.Home)
		case "ws_user":
			return []byte(vars.User)
		case "ws_server_root":
			return []byte(vars.ServerRoot)
		}

		if name, ok := strings.CutPrefix(token, secretsPrefix); ok {
			value, err := secret(name)
			if err != nil {
				failure = err
				return nil
			}

			return value
		}

		failure = fmt.Errorf("unknown template token ${%s}", token)
		return nil
	})

	if failure != nil {
		return nil, failure
	}

	return rendered, nil
}
