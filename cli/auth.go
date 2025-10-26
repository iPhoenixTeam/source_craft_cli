package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
    appConfigDir  = "srccli"
    configRelFile = "config.json"
)

type CLIConfig struct {
    AuthToken string            `json:"auth_token"`
    Settings  map[string]string `json:"settings"`
}

func DispatchAuth(command string, args []string) {
    switch command {
    case "login":
        fs := NewCmd("auth login", "Usage: %s auth login <token>\n", flag.ContinueOnError)
        if err := fs.Parse(args); err == nil {
            rem := Require(fs, 1, "Usage: auth login <token>")
            if rem == nil {
                return
            }
            AuthLogin(rem[0])
            return
        } else {
            return
        }
    case "logout":
        fs := NewCmd("auth logout", "Usage: %s auth logout\n", flag.ContinueOnError)
        if err := fs.Parse(args); err == nil {
            AuthLogout()
            return
        } else {
            return
        }
    case "--help", "-h", "help", "":
        fmt.Fprintln(os.Stderr, `auth commands:
  auth login <token>
  auth logout`)
        return
    default:
        fmt.Fprintln(os.Stderr, "unknown auth command:", command)
        return
    }
}

func DispatchConfig(command string, args []string) {
    switch command {
    case "set":
        fs := NewCmd("config set", "Usage: %s config set <key> <value>\n", flag.ContinueOnError)
        if err := fs.Parse(args); err == nil {
            rem := Require(fs, 2, "Usage: config set <key> <value>")
            if rem == nil {
                return
            }
            ConfigSet(rem[0], rem[1])
            return
        } else {
            return
        }
    case "get":
        fs := NewCmd("config get", "Usage: %s config get [<key>]\n", flag.ContinueOnError)
        if err := fs.Parse(args); err == nil {
            rem := Require(fs, 0, "")
            if rem == nil {
                return
            }
            key := ""
            if len(rem) > 0 {
                key = rem[0]
            }
            ConfigPrint(key)
            return
        } else {
            return
        }
    case "--help", "-h", "help", "":
        fmt.Fprintln(os.Stderr, `config commands:
  config set <key> <value>
  config get [<key>]`)
        return
    default:
        fmt.Fprintln(os.Stderr, "unknown config command:", command)
        return
    }
}

func AuthLogin(token string) {
    if token == "" {
        Ensure(fmt.Errorf("token is empty"))
    }
    cfg, err := loadConfig()
    Ensure(err)

    cfg.AuthToken = token

    if cfg.Settings == nil {
        cfg.Settings = make(map[string]string)
    }

    Ensure(saveConfig(cfg))
    fmt.Println("Logged in")
}

func AuthLogout() {
    cfg, err := loadConfig()
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Println("Already logged out")
            return
        }
        Ensure(err)
    }

    cfg.AuthToken = ""
    Ensure(saveConfig(cfg))
    fmt.Println("Logged out")
}

func ConfigSet(key, value string) {
    if key == "" {
        Ensure(fmt.Errorf("key is empty"))
    }
    cfg, err := loadConfig()
    Ensure(err)

    if cfg.Settings == nil {
        cfg.Settings = make(map[string]string)
    }
    cfg.Settings[key] = value

    Ensure(saveConfig(cfg))
    fmt.Printf("Config %s set to %s\n", key, value)
}

func ConfigPrint(key string) {
    cfg, err := loadConfig()
    Ensure(err)

    if key == "" {
        b, _ := json.MarshalIndent(cfg, "", "  ")
        fmt.Println(string(b))
        return
    }
    if cfg.Settings == nil {
        fmt.Println("")
        return
    }
    if v, ok := cfg.Settings[key]; ok {
        fmt.Println(v)
        return
    }

    fmt.Println("")
}

func configFilePath() (string, error) {
    dir, err := os.UserConfigDir()
    if err != nil {
        return "", err
    }
    appDir := filepath.Join(dir, appConfigDir)
    if err := os.MkdirAll(appDir, 0o700); err != nil {
        return "", err
    }
    return filepath.Join(appDir, configRelFile), nil
}

func loadConfig() (*CLIConfig, error) {
    path, err := configFilePath()
    if err != nil {
        return nil, err
    }
    f, err := os.Open(path)
    if err != nil {
        return &CLIConfig{Settings: make(map[string]string)}, err
    }
    defer f.Close()

    var cfg CLIConfig
    dec := json.NewDecoder(f)
    if err := dec.Decode(&cfg); err != nil {
        return &CLIConfig{Settings: make(map[string]string)}, err
    }
    if cfg.Settings == nil {
        cfg.Settings = make(map[string]string)
    }
    return &cfg, nil
}

func saveConfig(cfg *CLIConfig) error {
    path, err := configFilePath()
    if err != nil {
        return err
    }
    tmp := path + ".tmp"
    f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
    if err != nil {
        return err
    }
    enc := json.NewEncoder(f)
    enc.SetIndent("", "  ")
    if err := enc.Encode(cfg); err != nil {
        f.Close()
        _ = os.Remove(tmp)
        return err
    }
    if err := f.Close(); err != nil {
        _ = os.Remove(tmp)
        return err
    }
    return os.Rename(tmp, path)
}
