package cli

import (
	"fmt"
	"strings"
)

func TruncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func IndentString(s string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func StringContains(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || (IndexOfString(s, sub) >= 0))
}

func IndexOfString(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func ShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func IndentMultilineString(s string, indent int) string {
    if s == "" {
        return ""
    }
    pad := strings.Repeat(" ", indent)
    lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
    for i := range lines {
        lines[i] = pad + lines[i]
    }
    return strings.Join(lines, "\n")
}

func requireArgs(args []string, want int, usage string) {
    if len(args) < want {
        Ensure(fmt.Errorf("usage: %s", usage))
    }
}