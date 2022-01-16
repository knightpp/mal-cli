package widget

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

type Widget interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View(w io.Writer)
}

type WidgetStack struct {
	slice []Widget
}

func NewWidgetStack(ws ...Widget) WidgetStack {
	return WidgetStack{ws}
}

func (ws *WidgetStack) Push(w Widget) Widget {
	ws.slice = append(ws.slice, w)
	return w
}

func (ws *WidgetStack) Pop() Widget {
	w := ws.Peek()
	if w != nil {
		ws.slice = ws.slice[:len(ws.slice)-1]
	}
	return w
}

func (ws *WidgetStack) Peek() Widget {
	len := len(ws.slice)
	if len == 0 {
		return nil
	}
	return ws.slice[len-1]
}

func (ws *WidgetStack) Init() tea.Cmd {
	if w := ws.Peek(); w != nil {
		return w.Init()
	}
	return nil
}

func (ws *WidgetStack) Update(msg tea.Msg) tea.Cmd {
	if w := ws.Peek(); w != nil {
		return w.Update(msg)
	}
	return nil
}

func (ws *WidgetStack) View(w io.Writer) {
	if widget := ws.Peek(); widget != nil {
		widget.View(w)
	}
}
