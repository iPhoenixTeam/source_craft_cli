package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type JsonObject = map[string] any

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

func Require(fs *flag.FlagSet, want int, usage string) []string {
    rem := fs.Args()
    if len(rem) < want {
        if usage != "" {
            fmt.Println(usage)
        } else {
            fs.Usage()
        }
        os.Exit(0)
    }
    return rem
}

func MapStringField(m map[string]any, key, subkey string) string {
    if v, ok := m[key]; ok && v != nil {
        if mm, ok := v.(map[string]any); ok {
            if s, ok := mm[subkey].(string); ok {
                return s
            }
        }
    }
    return ""
}

func NumberFrom(v any) (int, bool) {
    switch n := v.(type) {
    case int:
        return n, true
    case int64:
        return int(n), true
    case float64:
        return int(n), true
    case float32:
        return int(n), true
    default:
        return 0, false
    }
}

func ParseDateFromMap(m map[string]any, keys ...string) string {
    for _, k := range keys {
        if v, ok := m[k]; ok && v != nil {
            if s, ok := v.(string); ok && s != "" {
                if t, err := time.Parse(time.RFC3339, s); err == nil {
                    return t.Format("2006-01-02")
                }
                return s
            }
        }
    }
    return ""
}

func ToString(s any) string {
	if s == nil {
        return ""
    }
    if v, ok := s.(string); ok && s != "" {
        return v
    }
    return ""
}

func prettyTime(v any) string {
    if v == nil {
        return ""
    }
    if s, ok := v.(string); ok && s != "" {
        if t, err := time.Parse(time.RFC3339, s); err == nil {
            return t.Format("2006-01-02")
        }
        return s
    }
    return ""
}