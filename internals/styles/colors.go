package styles

import catppuccin "github.com/catppuccin/go"

var frappe = catppuccin.Frappe

var (
	ColorText    = frappe.Text().Hex
	ColorSubtext = frappe.Subtext1().Hex

	ColorInfo    = frappe.Blue().Hex
	ColorSuccess = frappe.Green().Hex
	ColorWarning = frappe.Yellow().Hex
	ColorError   = frappe.Red().Hex
	ColorMuted   = frappe.Overlay0().Hex
	ColorAccent  = frappe.Teal().Hex
	ColorHeader  = frappe.Mauve().Hex
	ColorBorder  = frappe.Surface2().Hex
)
