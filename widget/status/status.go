package status

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knightpp/mal-cli/widget"
)

type Status int

const (
	Loading Status = iota + 1
	Idle
)

type Statuser interface {
	SetStatusLine(sl *StatusModel)
}

type ChangeStatusFn func(Status)

func (s Status) String() string {
	switch s {
	case Loading:
		return "Loading"
	case Idle:
		return "Idling"
	default:
		return "Unknown"
	}
}

type StatusModel struct {
	spinner spinner.Model
	status  Status
	body    widget.Widget
}

func NewStatusLine(body widget.Widget) *StatusModel {
	s := spinner.NewModel()
	s.Spinner = spinner.Points
	statusModel := &StatusModel{
		spinner: s,
		status:  Loading,
		body:    body,
	}
	if st, ok := body.(Statuser); ok {
		st.SetStatusLine(statusModel)
	}
	return statusModel
}

func (line *StatusModel) SetStatus(status Status) {
	line.status = status
}

func (line *StatusModel) Init() tea.Cmd {
	return tea.Batch(line.body.Init(), spinner.Tick)
}

func (line *StatusModel) Update(msg tea.Msg) tea.Cmd {
	if line.status == Loading {
		var cmd tea.Cmd
		line.spinner, cmd = line.spinner.Update(msg)
		return tea.Batch(cmd, line.body.Update(msg))
	}
	return line.body.Update(msg)
}

func (line *StatusModel) View(w io.Writer) {
	statusLineText := lipgloss.NewStyle().
		Background(lipgloss.Color("#343433")).
		Width(15).
		Bold(true).
		Align(lipgloss.Center)

	if line.status == Loading {
		fmt.Fprintf(w, "%s %s\n",
			statusLineText.Render(line.status.String()),
			line.spinner.View(),
		)
	} else {
		fmt.Fprintf(w, "%s\n", statusLineText.Render(line.status.String()))
	}
	line.body.View(w)
}
