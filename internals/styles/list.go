package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

func ListStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Text))
}

func ListEnumeratorStyle() lipgloss.Style {
	return Muted().PaddingRight(2)
}

func ListItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Text))
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
