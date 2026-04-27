package styles

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
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

type DescriptionItem struct {
	Name        string
	Description string
}

func DescriptionList(items []DescriptionItem) []any {
	maxLen := 0
	for _, item := range items {
		if len(item.Name) > maxLen {
			maxLen = len(item.Name)
		}
	}
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = Key().Width(maxLen).Render(item.Name) +
			Muted().Render(" — ") +
			Value().Render(item.Description)
	}
	return result
}
