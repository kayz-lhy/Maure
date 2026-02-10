package maurecobra

import (
	"fmt"
	"io"
	"strings"
)

func NormalizeLegacyArgs(rawArgs []string, warn io.Writer) []string {
	if len(rawArgs) == 0 {
		return nil
	}

	normalized := append([]string(nil), rawArgs...)
	warned := false

	for i, token := range normalized {
		if token != "parse-log" {
			continue
		}
		for j := i + 1; j < len(normalized); j++ {
			arg := normalized[j]
			if arg == "--" {
				break
			}
			if arg == "--format" {
				normalized[j] = "--log-format"
				if !warned && warn != nil {
					fmt.Fprintln(warn, "Warning: parse-log flag --format is deprecated; use --log-format.")
					warned = true
				}
				continue
			}
			if strings.HasPrefix(arg, "--format=") {
				normalized[j] = "--log-format=" + strings.TrimPrefix(arg, "--format=")
				if !warned && warn != nil {
					fmt.Fprintln(warn, "Warning: parse-log flag --format is deprecated; use --log-format.")
					warned = true
				}
			}
		}
		break
	}

	return normalized
}

func HasLegacyFlagOrder(rawArgs []string, cmdName string) bool {
	if len(rawArgs) == 0 {
		return false
	}

	idx := -1
	for i, token := range rawArgs {
		if token == cmdName {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(rawArgs) {
		return false
	}

	hasPositional := false
	for _, token := range rawArgs[idx+1:] {
		if token == "--" {
			break
		}
		if strings.HasPrefix(token, "-") {
			if hasPositional {
				return true
			}
			continue
		}
		hasPositional = true
	}

	return false
}
