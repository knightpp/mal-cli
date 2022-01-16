package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/knightpp/mal-api/mal"
	"github.com/knightpp/mal-cli/util"
)

type SearchModel struct {
	err error

	textInput textinput.Model

	searchAnimeList *mal.AnimeSearchPage
}

type MsgSearchAnime struct {
	List *mal.AnimeSearchPage
}

func (s *SearchModel) View() string {
	var buf strings.Builder
	buf.WriteString(s.textInput.View())
	if s.searchAnimeList == nil {
		return buf.String()
	}

	for _, anime := range s.searchAnimeList.Data {
		fmt.Fprintf(&buf, "%s", util.PrefTitle(&anime.Node))
	}
	return buf.String()
}

func (s *SearchModel) Update(msg tea.Msg) tea.Cmd {
	if s == nil {
		return nil
	}
	var cmd tea.Cmd
	s.textInput, cmd = s.textInput.Update(msg)
	return cmd
}

func NewSearch() (*SearchModel, tea.Cmd) {
	ti := textinput.NewModel()
	ti.Placeholder = "Anime name"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	return &SearchModel{
		textInput: ti,
	}, textinput.Blink
}
