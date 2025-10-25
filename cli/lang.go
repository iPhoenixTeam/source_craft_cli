package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

var (
	i18nMu       sync.RWMutex
	locales      = map[string]map[string]any{} // lang -> nested map
	activeLocale = "en"
)

func InitI18n() error {
	cfg, _ := loadConfig()
	if cfg != nil && cfg.Settings != nil {
		if loc, ok := cfg.Settings["locale"]; ok && loc != "" {
			activeLocale = loc
		}
	}
	dir, err := languagesDirPath()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(dir, 0o755)
	// prefer JSON files named like en.json, ru.json to follow GitHub-style locale files
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		ext := strings.ToLower(filepath.Ext(name))
		base := strings.TrimSuffix(name, ext)
		if ext == ".json" {
			_ = loadLocaleJSON(filepath.Join(dir, name), base)
		}
	}
	// ensure at least en exists (create minimal builtin if missing)
	if _, ok := locales["en"]; !ok {
		loadBuiltInEn()
	}
	return nil
}

func SetLocaleLocale(lang string) error {
	i18nMu.RLock()
	_, exists := locales[lang]
	i18nMu.RUnlock()
	if !exists {
		dir, err := languagesDirPath()
		if err != nil {
			return err
		}
		p := filepath.Join(dir, lang+".json")
		if _, err := os.Stat(p); err == nil {
			if err := loadLocaleJSON(p, lang); err != nil {
				return err
			}
		}
	}
	i18nMu.RLock()
	_, exists = locales[lang]
	i18nMu.RUnlock()
	if !exists {
		return fmt.Errorf("language %s not available", lang)
	}
	activeLocale = lang
	cfg, err := loadConfig()
	if err == nil && cfg != nil {
		if cfg.Settings == nil {
			cfg.Settings = map[string]string{}
		}
		cfg.Settings["locale"] = lang
		_ = saveConfig(cfg)
	}
	return nil
}

func loadLocaleJSON(path, lang string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	nested := map[string]any{}
	// convert flat dot keys to nested map (supports keys like "issue.title")
	for k, v := range raw {
		setNested(nested, k, v)
	}
	i18nMu.Lock()
	locales[lang] = nested
	i18nMu.Unlock()
	return nil
}

func setNested(m map[string]any, dotted string, val any) {
	parts := strings.Split(dotted, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = val
			return
		}
		if nxt, ok := cur[p]; ok {
			if mm, ok2 := nxt.(map[string]any); ok2 {
				cur = mm
				continue
			}
		}
		newm := map[string]any{}
		cur[p] = newm
		cur = newm
	}
}

func T(key string, data any) string {
	i18nMu.RLock()
	defer i18nMu.RUnlock()

	// lookup in active locale, then fallback to en
	if v, ok := getNested(locales[activeLocale], key); ok {
		return renderValue(v, data)
	}
	if v, ok := getNested(locales["en"], key); ok {
		return renderValue(v, data)
	}
	// fallback: return key (GitHub often shows key when missing)
	return key
}

func getNested(m map[string]any, dotted string) (any, bool) {
	if m == nil {
		return nil, false
	}
	parts := strings.Split(dotted, ".")
	cur := m
	for i, p := range parts {
		v, ok := cur[p]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			return v, true
		}
		if mm, ok := v.(map[string]any); ok {
			cur = mm
		} else {
			return nil, false
		}
	}
	return nil, false
}

func renderValue(v any, data any) string {
	switch t := v.(type) {
	case string:
		// support two interpolation styles:
		// 1) positional via fmt.Sprintf when data is []any
		// 2) named via Go text/template when data is map[string]any
		if data == nil {
			return t
		}
		switch d := data.(type) {
		case []any:
			// positional (like Minecraft {0} style can be expressed as %s/%d in strings)
			args := make([]any, len(d))
			for i := range d {
				args[i] = d[i]
			}
			return fmt.Sprintf(t, args...)
		case []string:
			args := make([]any, len(d))
			for i := range d {
				args[i] = d[i]
			}
			return fmt.Sprintf(t, args...)
		case map[string]any:
			// template with {{.name}} placeholders
			tpl, err := template.New("i18n").Option("missingkey=zero").Parse(t)
			if err != nil {
				return t
			}
			var sb strings.Builder
			_ = tpl.Execute(&sb, d)
			return sb.String()
		case map[string]string:
			// convert to map[string]any
			ctx := map[string]any{}
			for k, vv := range d {
				ctx[k] = vv
			}
			tpl, err := template.New("i18n").Option("missingkey=zero").Parse(t)
			if err != nil {
				return t
			}
			var sb strings.Builder
			_ = tpl.Execute(&sb, ctx)
			return sb.String()
		default:
			// fallback: try fmt
			return fmt.Sprintf(t, d)
		}
	default:
		// non-string resource: try marshal to string via fmt
		return fmt.Sprint(v)
	}
}

func loadBuiltInEn() {
	// minimal built-in english keys modeled after GitHub CLI style
	flat := map[string]any{
		"cli.title":   "src — command line for SourceCraft.dev",
		"cli.version": "Version: %s",
		"help.usage":  "Usage: src <command> [arguments]",
		"help.run":    "Run 'src %s --help' for details about a command.",
		"msg.logged_in":  "Logged in",
		"msg.logged_out": "Logged out",
	}
	nested := map[string]any{}
	for k, v := range flat {
		setNested(nested, k, v)
	}
	i18nMu.Lock()
	locales["en"] = nested
	i18nMu.Unlock()
}