package main

import (
    "errors"
    "fmt"
    "os"
    "sort"
    "strconv"
    "strings"
)

// Version of this library
const Version = "0.1.0"

// OptionType enumerates supported option value kinds.
type OptionType int

const (
    OptFlag OptionType = iota // boolean flag (presence)
    OptString                 // single string
    OptInt                    // single int
    OptFloat                  // single float64
    OptStringList             // multiple string values (repeatable)
)

// Option describes a single option.
type Option struct {
    // Names: short like "-v" (rune or string) or long like "--verbose"
    Short string // e.g. "v" (without "-")
    Long  string // e.g. "verbose" (without "--")

    Type OptionType

    Required bool   // must be present
    Default  any    // default value when absent
    Help     string // help text

    Validate func(values []string) error

    present bool
    values  []string
}

type Parser struct {
    optsByShort map[string]*Option
    optsByLong  map[string]*Option
    order       []*Option

    Args []string

    ProgramName string
    UsageText   string
    ShowHelp    bool
}

// NewParser creates a new Parser.
func NewParserWith(name string) *Parser {
    return &Parser{
        optsByShort: make(map[string]*Option),
        optsByLong:  make(map[string]*Option),
        order:       []*Option{},
        ProgramName: name,
    }
}

func NewParser() *Parser {
	return NewParserWith(filepathBase(os.Args[0]))
}

// ShortFlag registers a boolean flag (no argument): -v / --verbose
func (p *Parser) ShortFlag(short, long, help string) *Option {
    opt := &Option{Short: short, Long: long, Type: OptFlag, Help: help}
    p.register(opt)
    return opt
}

// StringOpt registers single string option: -n name / --name name
func (p *Parser) StringOpt(short, long, help string, def string) *Option {
    opt := &Option{Short: short, Long: long, Type: OptString, Help: help, Default: def}
    p.register(opt)
    return opt
}

// IntOpt registers single int option
func (p *Parser) IntOpt(short, long, help string, def int) *Option {
    opt := &Option{Short: short, Long: long, Type: OptInt, Help: help, Default: def}
    p.register(opt)
    return opt
}

// FloatOpt registers single float option
func (p *Parser) FloatOpt(short, long, help string, def float64) *Option {
    opt := &Option{Short: short, Long: long, Type: OptFloat, Help: help, Default: def}
    p.register(opt)
    return opt
}

// StringListOpt registers repeatable string option (can appear multiple times)
func (p *Parser) StringListOpt(short, long, help string) *Option {
    opt := &Option{Short: short, Long: long, Type: OptStringList, Help: help}
    p.register(opt)
    return opt
}

// register stores option and validates names
func (p *Parser) register(opt *Option) {
    if opt.Short != "" {
        if _, ok := p.optsByShort[opt.Short]; ok {
            panic("duplicate short option: " + opt.Short)
        }
        p.optsByShort[opt.Short] = opt
    }
    if opt.Long != "" {
        if _, ok := p.optsByLong[opt.Long]; ok {
            panic("duplicate long option: " + opt.Long)
        }
        p.optsByLong[opt.Long] = opt
    }
    p.order = append(p.order, opt)
}

func (p *Parser) Parse(argv []string) error {
    p.Args = nil
    i := 0
    for i < len(argv) {
        t := argv[i]

        if t == "--" {
            p.Args = append(p.Args, argv[i+1:]...)
            return p.postValidate()
        }
        if strings.HasPrefix(t, "--") {
            // long option: --name or --name=value
            nameVal := strings.TrimPrefix(t, "--")
            name, val, hasEq := splitNameValue(nameVal)
            opt := p.optsByLong[name]
            if opt == nil {
                return fmt.Errorf("unknown option --%s", name)
            }
            if err := p.consumeOption(opt, hasEq, val, &i, argv); err != nil {
                return err
            }
            continue
        }
        if strings.HasPrefix(t, "-") && t != "-" {
            // could be combined short flags like -abc or -nValue or -n Value
            shorts := strings.TrimPrefix(t, "-")
            // if option expects value and forms like -nVALUE (no space) we treat rest as value for first short that expects it.
            consumedInline := false
            for si := 0; si < len(shorts); si++ {
                s := string(shorts[si])
                opt := p.optsByShort[s]
                if opt == nil {
                    return fmt.Errorf("unknown option -%s", s)
                }

                if opt.Type != OptFlag && si < len(shorts)-1 {
                    val := shorts[si+1:]
                    if err := p.consumeOption(opt, true, val, &i, argv); err != nil {
                        return err
                    }
                    consumedInline = true
                    break
                }
                if err := p.consumeOption(opt, false, "", &i, argv); err != nil {
                    return err
                }
            }
            if consumedInline {
                // move to next argv
            }
            i++
            continue
        }
        // positional argument
        p.Args = append(p.Args, t)
        i++
    }
    return p.postValidate()
}

// consumeOption handles storing option values; i is pointer to current argv index.
func (p *Parser) consumeOption(opt *Option, hasValueInline bool, inlineVal string, i *int, argv []string) error {
    opt.present = true
    switch opt.Type {
    case OptFlag:
        // boolean true by presence
        opt.values = []string{"true"}
        return nil
    case OptString, OptInt, OptFloat:
        var val string
        if hasValueInline {
            val = inlineVal
        } else {
            // next argv must be value
            if *i+1 >= len(argv) {
                return fmt.Errorf("option requires a value: %s", optDisplay(opt))
            }
            *i = *i + 1
            val = argv[*i]
        }
        opt.values = []string{val}
        // run validation if present
        if opt.Validate != nil {
            if err := opt.Validate([]string{val}); err != nil {
                return fmt.Errorf("validation failed for %s: %w", optDisplay(opt), err)
            }
        }
        return nil
    case OptStringList:
        var val string
        if hasValueInline {
            val = inlineVal
        } else {
            if *i+1 >= len(argv) {
                return fmt.Errorf("option requires a value: %s", optDisplay(opt))
            }
            *i = *i + 1
            val = argv[*i]
        }
        opt.values = append(opt.values, val)
        if opt.Validate != nil {
            if err := opt.Validate(opt.values); err != nil {
                return fmt.Errorf("validation failed for %s: %w", optDisplay(opt), err)
            }
        }
        return nil
    default:
        return fmt.Errorf("unsupported option type for %s", optDisplay(opt))
    }
}

// postValidate enforces required options and applies defaults.
func (p *Parser) postValidate() error {
    // check required options
    for _, opt := range p.order {
        if opt.Required && !opt.present {
            return fmt.Errorf("required option missing: %s", optDisplay(opt))
        }
        if !opt.present && opt.Default != nil {
            // set default into values
            switch opt.Type {
            case OptFlag:
                if dv, ok := opt.Default.(bool); ok && dv {
                    opt.values = []string{"true"}
                    opt.present = true
                }
            case OptString:
                opt.values = []string{fmt.Sprint(opt.Default)}
            case OptInt:
                opt.values = []string{fmt.Sprint(opt.Default)}
            case OptFloat:
                opt.values = []string{fmt.Sprint(opt.Default)}
            case OptStringList:
                // assume Default is []string or string
                switch d := opt.Default.(type) {
                case []string:
                    opt.values = append([]string{}, d...)
                case string:
                    opt.values = []string{d}
                }
            }
        }
        // final validation if not run earlier
        if opt.present && opt.Validate != nil {
            if err := opt.Validate(opt.values); err != nil {
                return fmt.Errorf("validation failed for %s: %w", optDisplay(opt), err)
            }
        }
    }
    return nil
}

// Helpers to read typed values after parsing

// IsPresent returns true if option was provided (or default set)
func (p *Parser) IsPresent(opt *Option) bool { return opt != nil && opt.present }

// GetString returns single string value or def if absent.
func (p *Parser) GetString(opt *Option) string {
    if opt == nil {
        return ""
    }
    if len(opt.values) > 0 {
        return opt.values[0]
    }
    if s, ok := opt.Default.(string); ok {
        return s
    }
    return ""
}

// GetStringList returns slice of strings (maybe empty)
func (p *Parser) GetStringList(opt *Option) []string {
    if opt == nil {
        return nil
    }
    if len(opt.values) > 0 {
        return append([]string{}, opt.values...)
    }
    if arr, ok := opt.Default.([]string); ok {
        return append([]string{}, arr...)
    }
    return nil
}

// GetInt returns parsed int or def
func (p *Parser) GetInt(opt *Option) (int, error) {
    if opt == nil {
        return 0, errors.New("option nil")
    }
    s := p.GetString(opt)
    if s == "" {
        if v, ok := opt.Default.(int); ok {
            return v, nil
        }
        return 0, nil
    }
    n, err := strconv.Atoi(s)
    if err != nil {
        return 0, fmt.Errorf("cannot convert %q to int for %s: %w", s, optDisplay(opt), err)
    }
    return n, nil
}

// GetFloat returns parsed float64 or def
func (p *Parser) GetFloat(opt *Option) (float64, error) {
    if opt == nil {
        return 0, errors.New("option nil")
    }
    s := p.GetString(opt)
    if s == "" {
        if v, ok := opt.Default.(float64); ok {
            return v, nil
        }
        return 0, nil
    }
    f, err := strconv.ParseFloat(s, 64)
    if err != nil {
        return 0, fmt.Errorf("cannot convert %q to float for %s: %w", s, optDisplay(opt), err)
    }
    return f, nil
}

// optDisplay renders readable option name
func optDisplay(opt *Option) string {
    if opt == nil {
        return "<nil>"
    }
    parts := []string{}
    if opt.Short != "" {
        parts = append(parts, "-"+opt.Short)
    }
    if opt.Long != "" {
        parts = append(parts, "--"+opt.Long)
    }
    return strings.Join(parts, "/")
}

// splits "name=value" into name, value, present
func splitNameValue(s string) (string, string, bool) {
    if idx := strings.IndexByte(s, '='); idx >= 0 {
        return s[:idx], s[idx+1:], true
    }
    return s, "", false
}

// HelpText generates usage/help text for registered options.
func (p *Parser) HelpText() string {
    lines := []string{}
    header := fmt.Sprintf("Usage: %s %s", p.ProgramName, strings.TrimSpace(p.UsageText))
    lines = append(lines, header)
    lines = append(lines, "")
    lines = append(lines, "Options:")
    // stable order
    opts := append([]*Option{}, p.order...)
    sort.Slice(opts, func(i, j int) bool {
        li, lj := opts[i].Long, opts[j].Long
        return li < lj
    })
    for _, o := range opts {
        names := []string{}
        if o.Short != "" {
            names = append(names, "-"+o.Short)
        }
        if o.Long != "" {
            names = append(names, "--"+o.Long)
        }
        name := strings.Join(names, ", ")
        typ := ""
        switch o.Type {
        case OptFlag:
            typ = ""
        case OptString:
            typ = " <string>"
        case OptInt:
            typ = " <int>"
        case OptFloat:
            typ = " <float>"
        case OptStringList:
            typ = " <value>..."
        }
        def := ""
        if o.Default != nil {
            def = fmt.Sprintf(" (default: %v)", o.Default)
        }
        req := ""
        if o.Required {
            req = " [required]"
        }
        lines = append(lines, fmt.Sprintf("  %-20s %s%s%s", name, typ, def, req))
        if o.Help != "" {
            lines = append(lines, fmt.Sprintf("      %s", o.Help))
        }
    }
    return strings.Join(lines, "\n")
}

// PrintHelp writes help to stdout.
func (p *Parser) PrintHelp() {
    fmt.Println(p.HelpText())
}

// filepathBase - simple base name
func filepathBase(p string) string {
    if p == "" {
        return ""
    }
    parts := strings.Split(p, string(os.PathSeparator))
    return parts[len(parts)-1]
}