package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

func ListStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)),
	)
}

func ListEnumeratorStyle() lipgloss.Style {
	return WithColor(Muted()).PaddingRight(2)
}

func ListItemStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)),
	)
}

func List(items ...any) *list.List {
	l := list.New(items...)
	l.Enumerator(list.Bullet)
	l.EnumeratorStyle(ListEnumeratorStyle())
	l.ItemStyle(ListItemStyle())

	return l
}

func NumberedList(items ...any) *list.List {
	return List(items...).Enumerator(list.Arabic)
}
