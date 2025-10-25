package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const cliVersion = "0.0.1"

var commands = map[string]string{
	"--help":                          "общая справка",
	"--version":                       "версия CLI",
	"repo list":                       "просмотр списка репозиториев пользователя",
	"repo create <name>":              "создание нового репозитория",
	"repo view <repo>":                "просмотр информации о репозитории",
	"repo clone <repo>":               "клонирование репозитория",
	"repo fork <repo>":                "создание форка репозитория",
	"repo sync":                       "синхронизация форка с оригинальным репозиторием",
	"pr list":                         "список pull requests",
	"pr create":                       "создание pull request",
	"pr view <id>":                    "просмотр деталей pull request",
	"pr merge <id>":                   "слияние pull request",
	"issue list":                      "просмотр списка задач",
	"issue create":                    "создание задачи",
	"issue view <id>":                 "просмотр данных задачи",
	"issue update <id>":               "обновление задачи",
	"issue close <id>":                "закрытие задачи",
	"milestone list":                  "список вех проекта",
	"milestone create":                "создание новой вехи",
	"milestone view <id>":             "просмотр деталей вехи",
	"stats repo":                      "статистика репозитория",
	"stats user":                      "статистика активности пользователя",
	"stats security":                  "статистика состояния безопасности",
	"report security":                 "сводный отчет по безопасности с приоритизированными рисками",
	"code search <query>":             "поиск по коду в репозитории",
	"policy view":                     "просмотр политик веток",
	"policy update":                   "обновление политик веток",
	"review-rules view":               "просмотр правил ревью кода",
	"review-rules update":             "обновление правил ревью",
	"auth login":                      "аутентификация через персональный токен",
	"auth logout":                     "выход из системы",
	"config set <key> <value>":        "настройка параметров CLI",
	"config get <key>":                "просмотр настроек",
	"security scan":                   "запуск сканирования безопасности",
	"security list":                   "просмотр результатов сканирования безопасности",
	"security view <id>":              "просмотр детали уязвимости",
	"dependencies scan":               "анализ зависимостей на уязвимости",
	"secrets scan":                    "сканирование на утечки секретов",
	"secrets list":                    "просмотр найденных секретов",
	"secrets resolve <id>":            "разрешение найденного секрета",
	"env create <name>":               "создание переменной окружения",
	"env list":                        "список переменных окружения",
	"env delete <name>":               "удаление переменной окружения",
	"package list":                    "просмотр пакетов в реестре",
	"package publish":                 "публикация пакета",
	"mirror create <repo>":            "создание зеркала репозитория",
	"mirror sync <repo>":              "синхронизация зеркала",
	"import <repo>":                   "импорт репозитория из GitHub",
	"workflow list":                   "список CI workflows",
	"workflow status":                 "статус workflow",
	"workflow logs <id>":              "вывод логов workflow",
	"workflow run <id>":               "запуск workflow",
	"access user <action>":            "управление пользователями доступа",
	"access invite <email>":           "приглашение пользователя",
	"access role <user> <role>":       "назначение роли пользователю",
}

func Help(args []string) {
	// if user asked for specific command help: src <command> --help
	if len(args) >= 1 && args[0] != "" && !(len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		cmd := strings.Join(args, " ")
		HelpCommand(cmd)
		return
	}
	PrintGeneralHelp()
}

func PrintGeneralHelp() {
	fmt.Println("src — командная утилита для SourceCraft.dev")
	fmt.Printf("Version: %s\n\n", cliVersion)

	fmt.Println("Usage:")
	fmt.Println("  src <command> [arguments]\n")
	fmt.Println("Commands:")
	// print commands sorted
	keys := make([]string, 0, len(commands))
	for k := range commands {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// decide column widths
	maxCmdLen := 0
	for _, k := range keys {
		if l := len(k); l > maxCmdLen {
			maxCmdLen = l
		}
	}
	if maxCmdLen < 20 {
		maxCmdLen = 20
	}

	for _, k := range keys {
		fmt.Printf("  %-*s  %s\n", maxCmdLen, k, commands[k])
	}

	fmt.Println("\nFlags:")
	fmt.Println("  --help        Show help for src or a specific command")
	fmt.Println("  --version     Print CLI version")
	fmt.Println("\nRun 'src <command> --help' for details about a command.")
}

func HelpCommand(cmd string) {
	// try to match exact key or partial matching by prefix
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		PrintGeneralHelp()
		return
	}

	// direct match
	if desc, ok := commands[cmd]; ok {
		printCommandHelp(cmd, desc)
		return
	}

	// try find candidates by prefix
	cands := map[string]string{}
	for k, v := range commands {
		if strings.HasPrefix(k, cmd) || strings.Contains(k, cmd) {
			cands[k] = v
		}
	}

	if len(cands) == 0 {
		// try split first token
		tokens := strings.Fields(cmd)
		if len(tokens) > 0 {
			prefix := tokens[0]
			for k, v := range commands {
				if strings.HasPrefix(k, prefix+" ") || k == prefix {
					cands[k] = v
				}
			}
		}
	}

	if len(cands) == 1 {
		for k, v := range cands {
			printCommandHelp(k, v)
			return
		}
	}

	if len(cands) > 1 {
		fmt.Printf("Help for '%s' — possible commands:\n\n", cmd)
		keys := make([]string, 0, len(cands))
		for k := range cands {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		max := 0
		for _, k := range keys {
			if l := len(k); l > max {
				max = l
			}
		}
		if max < 16 {
			max = 16
		}
		for _, k := range keys {
			fmt.Printf("  %-*s  %s\n", max, k, cands[k])
		}
		fmt.Println("\nRun 'src " + keys[0] + " --help' for the selected command.")
		return
	}

	fmt.Printf("No help found for '%s'\n", cmd)
}

func printCommandHelp(cmd, desc string) {
	fmt.Printf("Usage: src %s\n\n", cmd)
	fmt.Printf("%s\n\n", desc)

	// add brief examples / expanded help for common commands
	switch {
	case strings.HasPrefix(cmd, "repo "):
		fmt.Println("Examples:")
		fmt.Println("  src repo list")
		fmt.Println("  src repo create my-repo")
		fmt.Println("  src repo view org/my-repo")
	case strings.HasPrefix(cmd, "issue "):
		fmt.Println("Examples:")
		fmt.Println("  src issue list")
		fmt.Println("  src issue create org repo \"Issue title\"")
		fmt.Println("  src issue view org repo issue-slug")
	case strings.HasPrefix(cmd, "auth "):
		fmt.Println("Examples:")
		fmt.Println("  src auth login <personal-token>")
		fmt.Println("  src auth logout")
	case strings.HasPrefix(cmd, "config "):
		fmt.Println("Examples:")
		fmt.Println("  src config set editor vim")
		fmt.Println("  src config get editor")
	case strings.HasPrefix(cmd, "code search"):
		fmt.Println("Examples:")
		fmt.Println("  src code search \"TODO\"")
		fmt.Println("  src code search \"func main\"")
	case strings.HasPrefix(cmd, "stats ") || strings.HasPrefix(cmd, "report "):
		fmt.Println("Examples:")
		fmt.Println("  src stats repo org repo")
		fmt.Println("  src report security org repo --top 10")
	default:
		// generic hint
		fmt.Println("Run 'src --help' to see list of available commands.")
	}

	// print flag hint
	fmt.Println("\nFlags:")
	fmt.Println("  --help     Show this help")
	fmt.Println("  --verbose  Enable verbose output")
}

func Version() {
	fmt.Printf("src %s\n", cliVersion)
}

// Entrypoint helper to wire to os.Args from main
func HandleHelpFromArgs() {
	args := os.Args[1:]
	// if no args or asked for global help/version
	if len(args) == 0 {
		PrintGeneralHelp()
		return
	}
	// --version
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		Version()
		return
	}
	// find if last token is --help
	if args[len(args)-1] == "--help" || args[len(args)-1] == "-h" {
		// drop the --help token and call Help with remaining tokens
		l := len(args)
		Help(args[:l-1])
		return
	}
	// otherwise not a help invocation
	PrintGeneralHelp()
}