package styles

import (
	"os"
	"slices"
	"strings"
)

func HelpTemplate() string {
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

func UsageTemplate() string {
	template := `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID .ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

	return colorizeTemplate(template)
}

func colorizeTemplate(template string) string {
	colorEnabled := ColorEnabled

	if _, exists := os.LookupEnv("WS_LOGGING_NO_COLOR"); exists {
		colorEnabled = false
	}

	if slices.Contains(os.Args, "--no-color") {
		colorEnabled = false
	}

	if !colorEnabled {
		return template
	}

	replacements := []struct{ old, new string }{
		{"Global Flags:", Badge().Render("Global Flags")},
		{"Flags:", SuccessBadge().Render("Flags")},
		{"Available Commands:", TipBadge().Render("Available Commands")},
		{"Additional Commands:", TipBadge().Render("Additional Commands")},
		{"Additional help topics:", Header().Render("Additional help topics:")},
		{`Use "{{.CommandPath}} [command] --help"`, "Use " + Code().Render("{{.CommandPath}} [command] --help")},
		{"Usage:", InfoBadge().Render("Usage")},
		{"Aliases:", SuccessBadge().Render("Aliases")},
		{"Examples:", WarningBadge().Render("Examples")},
	}

	result := template
	for _, replacement := range replacements {
		result = strings.ReplaceAll(result, replacement.old, replacement.new)
	}

	return result
}
