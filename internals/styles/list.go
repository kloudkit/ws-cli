package styles

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/list"
)

func ListStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Text)
}

func ListEnumeratorStyle() lipgloss.Style {
	return Muted().PaddingRight(2).MarginLeft(2)
}

func ListItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Text)
}

func List(items ...any) *list.List {
	return list.New(items...).
		Enumerator(list.Bullet).
		EnumeratorStyle(ListEnumeratorStyle()).
		ItemStyle(ListItemStyle())
}

func NumberedList(items ...any) *list.List {
	return List(items...).Enumerator(list.Arabic)
}
