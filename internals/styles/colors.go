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

	BgSuccess = frappe.Green().Hex
	BgWarning = frappe.Yellow().Hex
	BgError   = frappe.Red().Hex
	BgInfo    = frappe.Blue().Hex
	BgMuted   = frappe.Surface0().Hex
	BgAccent  = frappe.Surface1().Hex

	ColorBase = frappe.Base().Hex
)
