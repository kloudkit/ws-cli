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

func hex(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

var CatppuccinFrappeStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "",
			Color:       new(hex(Text)),
		},
		Margin: new(uint(glamourMargin)),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  new(hex(Yellow)),
			Italic: new(true),
		},
		Indent:      new(uint(1)),
		IndentToken: new("│ "),
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: new(hex(Text))},
		},
		LevelIndent: glamourListLevelStep,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       new(hex(Mauve)),
			Bold:        new(true),
		},
	},
	H1: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "# "}},
	H2: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "## "}},
	H3: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "### "}},
	H4: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "#### "}},
	H5: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "##### "}},
	H6: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "###### "}},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: new(true),
	},
	Emph: ansi.StylePrimitive{
		Color:  new(hex(Peach)),
		Italic: new(true),
	},
	Strong: ansi.StylePrimitive{
		Color: new(hex(Mauve)),
		Bold:  new(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  new(hex(Overlay0)),
		Format: "\n────\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       new(hex(Blue)),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     new(hex(Blue)),
		Underline: new(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: new(hex(Teal)),
	},
	Image: ansi.StylePrimitive{
		Color:     new(hex(Blue)),
		Underline: new(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  new(hex(Teal)),
		Format: "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           new(hex(Teal)),
			BackgroundColor: new(hex(Surface1)),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: new(hex(Text)),
			},
			Margin: new(uint(glamourMargin)),
		},
		Chroma: &ansi.Chroma{
			Text:                ansi.StylePrimitive{Color: new(hex(Text))},
			Error:               ansi.StylePrimitive{Color: new(hex(Text)), BackgroundColor: new(hex(Red))},
			Comment:             ansi.StylePrimitive{Color: new(hex(Overlay0))},
			CommentPreproc:      ansi.StylePrimitive{Color: new(hex(Teal))},
			Keyword:             ansi.StylePrimitive{Color: new(hex(Mauve))},
			KeywordReserved:     ansi.StylePrimitive{Color: new(hex(Mauve))},
			KeywordNamespace:    ansi.StylePrimitive{Color: new(hex(Mauve))},
			KeywordType:         ansi.StylePrimitive{Color: new(hex(Yellow))},
			Operator:            ansi.StylePrimitive{Color: new(hex(Teal))},
			Punctuation:         ansi.StylePrimitive{Color: new(hex(Overlay1))},
			Name:                ansi.StylePrimitive{Color: new(hex(Blue))},
			NameConstant:        ansi.StylePrimitive{Color: new(hex(Peach))},
			NameBuiltin:         ansi.StylePrimitive{Color: new(hex(Red))},
			NameTag:             ansi.StylePrimitive{Color: new(hex(Mauve))},
			NameAttribute:       ansi.StylePrimitive{Color: new(hex(Yellow))},
			NameClass:           ansi.StylePrimitive{Color: new(hex(Yellow))},
			NameDecorator:       ansi.StylePrimitive{Color: new(hex(Blue))},
			NameFunction:        ansi.StylePrimitive{Color: new(hex(Blue))},
			LiteralNumber:       ansi.StylePrimitive{Color: new(hex(Peach))},
			LiteralString:       ansi.StylePrimitive{Color: new(hex(Green))},
			LiteralStringEscape: ansi.StylePrimitive{Color: new(hex(Teal))},
			GenericDeleted:      ansi.StylePrimitive{Color: new(hex(Red))},
			GenericEmph:         ansi.StylePrimitive{Italic: new(true)},
			GenericInserted:     ansi.StylePrimitive{Color: new(hex(Green))},
			GenericStrong:       ansi.StylePrimitive{Bold: new(true)},
			GenericSubheading:   ansi.StylePrimitive{Color: new(hex(Subtext1))},
			Background:          ansi.StylePrimitive{BackgroundColor: new(hex(Base))},
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
