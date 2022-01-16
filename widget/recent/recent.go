package recent

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knightpp/mal-api/mal"
	"github.com/knightpp/mal-cli/util"
	"github.com/knightpp/mal-cli/widget/status"
	"go.uber.org/zap"
)

type MsgAppendToMyRecentList struct {
	Ok  *mal.UserAnimePage
	Err error
}

type MsgLoadedMyRecentAnimeList struct {
	List *mal.UserAnimePage
}

type RecentModel struct {
	log        *zap.Logger
	client     *mal.Client
	list       *mal.UserAnimePage
	statusLine *status.StatusModel

	selected int

	width  int
	height int
}

type MsgUpdatedMyRecentAnimeList struct {
	Ok   *mal.AnimeUpdateResponse
	List *mal.UserAnime
	Err  error
}

func New(client *mal.Client) *RecentModel {
	return &RecentModel{
		client:     client,
		statusLine: nil,
		selected:   0,
	}
}

func (m *RecentModel) Init() tea.Cmd {
	m.statusLine.SetStatus(status.Loading)
	return func() tea.Msg {
		list, err := m.client.GetMyAnimeList(mal.AnimeListOptions{
			Status: mal.StatusWatching,
			Sort:   mal.SortListUpdatedAt,
		}, mal.Fields("list_status", "num_episodes", "alternative_titles"))

		if err != nil {
			return tea.Quit
		}
		return MsgLoadedMyRecentAnimeList{List: list}
	}
}

func (m *RecentModel) Update(msg tea.Msg) tea.Cmd {
	if m == nil {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "w":
			if m.list == nil {
				break
			}
			if m.selected-1 >= 0 {
				m.selected -= 1
				return nil
			}
		case "s":
			if m.list == nil {
				break
			}
			if m.selected < len(m.list.Data)-1 {
				m.selected += 1
				return nil
			} else {
				m.statusLine.SetStatus(status.Loading)

				return tea.Batch(
					func() tea.Msg {
						list, err := m.list.NextPage(m.client)
						return MsgAppendToMyRecentList{
							Ok:  list,
							Err: err,
						}
					},
				)
			}
		case "r":

		case "/":
			panic("TODO")
			// if m.state == stateMyRecentAnime {
			// 	m.state = stateSearchAnime
			// 	m.statusline.SetStatus(status.Idle)
			// 	s, cmd := NewSearch()
			// 	m.search = s
			// 	return m, cmd
			// }
		case "d":
			m.statusLine.SetStatus(status.Loading)
			return m.addNumWatchedEpisodes(1)
		case "a":
			m.statusLine.SetStatus(status.Loading)
			return m.addNumWatchedEpisodes(-1)
		}

	case MsgLoadedMyRecentAnimeList:
		m.list = msg.List
		m.statusLine.SetStatus(status.Idle)
		return nil
	case MsgUpdatedMyRecentAnimeList:
		if msg.Err != nil {
			return tea.Quit
		}
		m.statusLine.SetStatus(status.Idle)
		msg.List.ListStatus.NumEpisodesWatched = msg.Ok.NumEpisodesWatched
		if msg.Ok.Status != "" {
			msg.List.ListStatus.Status = msg.Ok.Status
		}
		return nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return nil
	case MsgAppendToMyRecentList:
		if msg.Err != nil {
			m.log.Error("MsgAppendToMyRecentList", zap.Error(msg.Err))
			return tea.Quit
		}
		m.list.Data = append(m.list.Data, msg.Ok.Data...)
		m.list.Paging.Next = msg.Ok.Paging.Next
		m.statusLine.SetStatus(status.Idle)
	}

	return nil
}

func (m *RecentModel) View(buf io.Writer) {
	if m == nil {
		fmt.Fprint(buf, "MyRecentModel is nil")
		return
	}
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Width(m.width - 10)
	watchedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("45")).
		Width(3).
		Align(lipgloss.Right)
	totalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Width(3).
		Align(lipgloss.Right)

	if m.list == nil {
		fmt.Fprint(buf, "recent::list is nil")
		return
	}
	if m.list.Paging.HasPrev() {
		fmt.Fprintf(buf, "   Previous ^^^")
	}
	for i, anime := range m.list.Data {
		if i == m.selected {
			fmt.Fprint(buf, "👉 ")
		} else {
			fmt.Fprint(buf, "   ")
		}
		fmt.Fprintf(buf, "%s / %s %s\n",
			watchedStyle.Render(fmt.Sprint(anime.ListStatus.NumEpisodesWatched)),
			totalStyle.Render(fmt.Sprint(anime.Node.NumEpisodes)),
			titleStyle.Render(util.LongTitle(&anime.Node)),
		)
	}
	if m.list.Paging.HasNext() {
		fmt.Fprintf(buf, "   Next vvv")
	}
}

func (recent *RecentModel) addNumWatchedEpisodes(n int) func() tea.Msg {
	list := &recent.list.Data[recent.selected]
	opts := mal.UpdateListOpts()
	episodes := list.Node.NumEpisodes
	watched := list.ListStatus.NumEpisodesWatched

	switch {
	case episodes == watched+n:
		opts.Status(mal.StatusCompleted)
		opts.NumWatchedEpisodes(episodes)
	case episodes > watched+n && watched+n >= 0:
		opts.NumWatchedEpisodes(watched + n)
	default:
		return nil
	}

	cmd := func() tea.Msg {
		resp, err := recent.client.UpdateAnimeList(list.Node.ID, opts)
		return MsgUpdatedMyRecentAnimeList{Ok: resp, Err: err, List: list}
	}
	return tea.Batch(cmd, spinner.Tick)
}

func (recent *RecentModel) SetStatusLine(sl *status.StatusModel) {
	recent.statusLine = sl
}
