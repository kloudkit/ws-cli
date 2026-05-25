package styles

import (
	"fmt"
	"image/color"

	"charm.land/glamour/v2/ansi"
)

const (
	glamourMargin        = 2
	glamourListLevelStep = 2
)

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func uintPtr(u uint) *uint       { return &u }

func hexPtr(c color.RGBA) *string {
	s := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	return &s
}

var CatppuccinFrappeStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "",
			Color:       hexPtr(Text),
		},
		Margin: uintPtr(glamourMargin),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  hexPtr(Yellow),
			Italic: boolPtr(true),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr("│ "),
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: hexPtr(Text)},
		},
		LevelIndent: glamourListLevelStep,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       hexPtr(Mauve),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "# "}},
	H2: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "## "}},
	H3: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "### "}},
	H4: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "#### "}},
	H5: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "##### "}},
	H6: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "###### "}},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
	},
	Emph: ansi.StylePrimitive{
		Color:  hexPtr(Peach),
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Color: hexPtr(Mauve),
		Bold:  boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  hexPtr(Overlay0),
		Format: "\n────\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       hexPtr(Blue),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     hexPtr(Blue),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: hexPtr(Teal),
	},
	Image: ansi.StylePrimitive{
		Color:     hexPtr(Blue),
		Underline: boolPtr(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  hexPtr(Teal),
		Format: "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           hexPtr(Teal),
			BackgroundColor: hexPtr(Surface1),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: hexPtr(Text),
			},
			Margin: uintPtr(glamourMargin),
		},
		Chroma: &ansi.Chroma{
			Text:                ansi.StylePrimitive{Color: hexPtr(Text)},
			Error:               ansi.StylePrimitive{Color: hexPtr(Text), BackgroundColor: hexPtr(Red)},
			Comment:             ansi.StylePrimitive{Color: hexPtr(Overlay0)},
			CommentPreproc:      ansi.StylePrimitive{Color: hexPtr(Teal)},
			Keyword:             ansi.StylePrimitive{Color: hexPtr(Mauve)},
			KeywordReserved:     ansi.StylePrimitive{Color: hexPtr(Mauve)},
			KeywordNamespace:    ansi.StylePrimitive{Color: hexPtr(Mauve)},
			KeywordType:         ansi.StylePrimitive{Color: hexPtr(Yellow)},
			Operator:            ansi.StylePrimitive{Color: hexPtr(Teal)},
			Punctuation:         ansi.StylePrimitive{Color: hexPtr(Overlay1)},
			Name:                ansi.StylePrimitive{Color: hexPtr(Blue)},
			NameConstant:        ansi.StylePrimitive{Color: hexPtr(Peach)},
			NameBuiltin:         ansi.StylePrimitive{Color: hexPtr(Red)},
			NameTag:             ansi.StylePrimitive{Color: hexPtr(Mauve)},
			NameAttribute:       ansi.StylePrimitive{Color: hexPtr(Yellow)},
			NameClass:           ansi.StylePrimitive{Color: hexPtr(Yellow)},
			NameDecorator:       ansi.StylePrimitive{Color: hexPtr(Blue)},
			NameFunction:        ansi.StylePrimitive{Color: hexPtr(Blue)},
			LiteralNumber:       ansi.StylePrimitive{Color: hexPtr(Peach)},
			LiteralString:       ansi.StylePrimitive{Color: hexPtr(Green)},
			LiteralStringEscape: ansi.StylePrimitive{Color: hexPtr(Teal)},
			GenericDeleted:      ansi.StylePrimitive{Color: hexPtr(Red)},
			GenericEmph:         ansi.StylePrimitive{Italic: boolPtr(true)},
			GenericInserted:     ansi.StylePrimitive{Color: hexPtr(Green)},
			GenericStrong:       ansi.StylePrimitive{Bold: boolPtr(true)},
			GenericSubheading:   ansi.StylePrimitive{Color: hexPtr(Subtext1)},
			Background:          ansi.StylePrimitive{BackgroundColor: hexPtr(Base)},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
		},
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n→ ",
	},
}
