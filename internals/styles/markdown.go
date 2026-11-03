package styles

import (
	"fmt"
	"image/color"
	"io"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
)

var trailingPad = regexp.MustCompile(`(?m)(?:\x1b\[[\d;]*m|[ \t])+$`)

func hex(c color.RGBA) *string {
	return new(fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
}

func fg(c color.RGBA) ansi.StylePrimitive {
	return ansi.StylePrimitive{Color: hex(c)}
}

func prefix(p string) ansi.StyleBlock {
	return ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: p}}
}

var markdownMargin = new(uint(2))

var MarkdownStyle = ansi.StyleConfig{
	Document:   ansi.StyleBlock{StylePrimitive: fg(Text), Margin: markdownMargin},
	BlockQuote: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: hex(Yellow), Italic: new(true)}, Indent: new(uint(1)), IndentToken: new("│ ")},
	List:       ansi.StyleList{StyleBlock: ansi.StyleBlock{StylePrimitive: fg(Text)}, LevelIndent: 2},
	Heading:    ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n", Color: hex(Mauve), Bold: new(true)}},
	H1:         prefix("# "),
	H2:         prefix("## "),
	H3:         prefix("### "),
	H4:         prefix("#### "),
	H5:         prefix("##### "),
	H6:         prefix("###### "),

	Strikethrough:  ansi.StylePrimitive{CrossedOut: new(true)},
	Emph:           ansi.StylePrimitive{Color: hex(Peach), Italic: new(true)},
	Strong:         ansi.StylePrimitive{Color: hex(Mauve), Bold: new(true)},
	HorizontalRule: ansi.StylePrimitive{Color: hex(Overlay0), Format: "\n────\n"},
	Item:           ansi.StylePrimitive{BlockPrefix: "• "},
	Enumeration:    ansi.StylePrimitive{BlockPrefix: ". ", Color: hex(Blue)},
	Task:           ansi.StyleTask{Ticked: "[✓] ", Unticked: "[ ] "},
	Link:           ansi.StylePrimitive{Color: hex(Blue), Underline: new(true)},
	LinkText:       fg(Teal),
	Image:          ansi.StylePrimitive{Color: hex(Blue), Underline: new(true)},
	ImageText:      ansi.StylePrimitive{Color: hex(Teal), Format: "Image: {{.text}} →"},
	Code:           ansi.StyleBlock{StylePrimitive: fg(Teal)},

	DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n→ "},

	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{StylePrimitive: fg(Text), Margin: markdownMargin},
		Chroma: &ansi.Chroma{
			Text:                fg(Text),
			Error:               ansi.StylePrimitive{Color: hex(Text), BackgroundColor: hex(Red)},
			Comment:             fg(Overlay0),
			CommentPreproc:      fg(Teal),
			Keyword:             fg(Mauve),
			KeywordReserved:     fg(Mauve),
			KeywordNamespace:    fg(Mauve),
			KeywordType:         fg(Yellow),
			Operator:            fg(Teal),
			Punctuation:         fg(Overlay1),
			Name:                fg(Blue),
			NameConstant:        fg(Peach),
			NameBuiltin:         fg(Red),
			NameTag:             fg(Mauve),
			NameAttribute:       fg(Yellow),
			NameClass:           fg(Yellow),
			NameDecorator:       fg(Blue),
			NameFunction:        fg(Blue),
			LiteralNumber:       fg(Peach),
			LiteralString:       fg(Green),
			LiteralStringEscape: fg(Teal),
			GenericDeleted:      fg(Red),
			GenericEmph:         ansi.StylePrimitive{Italic: new(true)},
			GenericInserted:     fg(Green),
			GenericStrong:       ansi.StylePrimitive{Bold: new(true)},
			GenericSubheading:   fg(Subtext1),
			Background:          ansi.StylePrimitive{BackgroundColor: hex(Base)},
		},
	},
}

func RenderMarkdown(w io.Writer, src string) error {
	if strings.TrimSpace(src) == "" {
		return nil
	}

	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(MarkdownStyle))
	if err != nil {
		return err
	}

	rendered, err := renderer.Render(src)
	if err != nil {
		return err
	}

	rendered = trailingPad.ReplaceAllStringFunc(rendered, func(pad string) string {
		if strings.Contains(pad, "\x1b") {
			return "\x1b[m"
		}
		return ""
	})
	_, err = io.WriteString(w, strings.Trim(rendered, "\n")+"\n")

	return err
}
