package initialize

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

type InitModel struct {
}

func New() *InitModel {
	return &InitModel{}
}

func (im *InitModel) Init() tea.Cmd {
	return nil
}

func (im *InitModel) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (im *InitModel) View(w io.Writer) {
	fmt.Fprint(w, "Initializing...")
}
