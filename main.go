package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/knightpp/mal-api/mal"
)

type status int

const (
	statusLoading status = iota
	statusIdle
)

func (s status) String() string {
	switch s {
	case statusLoading:
		return "Loading..."
	case statusIdle:
		return "Idling"
	default:
		return "unknown"
	}
}

type model struct {
	spinner spinner.Model
	status  status

	list     *mal.UserAnimeListPage
	selected int
	client   *mal.Client
}

func initialModel() model {
	s := spinner.NewModel()
	s.Spinner = spinner.Points

	return model{
		spinner:  s,
		selected: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(spinner.Tick, tea.Cmd(func() tea.Msg {
		client, err := setupClient()
		if err != nil {
			return MsgLoadedAnimeList{Err: err}
		}

		list, err := client.GetMyAnimeList(mal.AnimeListOptions{
			Status: mal.StatusWatching,
			Sort:   mal.SortListUpdatedAt,
		}, mal.Fields("list_status", "num_episodes", "alternative_titles"))

		return MsgLoadedAnimeList{Ok: list, Client: client, Err: err}
	}))
}

type MsgLoadedAnimeList struct {
	Ok     *mal.UserAnimeListPage
	Client *mal.Client
	Err    error
}

type MsgUpdatedList struct {
	Ok   *mal.AnimeUpdateResponse
	List *mal.UserAnimeList
	Err  error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		case "w":
			if m.list == nil {
				break
			}
			if m.selected-1 >= 0 {
				m.selected -= 1
				return m, nil
			}
		case "s":
			if m.list == nil {
				break
			}
			if m.selected+1 <= len(m.list.Data) {
				m.selected += 1
				return m, nil
			}

		case "d":
			m.status = statusLoading
			return m, addNumWatchedEpisodes(1, m)
		case "a":
			m.status = statusLoading
			return m, addNumWatchedEpisodes(-1, m)
		}
	case MsgLoadedAnimeList:
		if msg.Err != nil {
			return m, tea.Quit
		}
		m.list = msg.Ok
		m.client = msg.Client
		m.status = statusIdle
		return m, nil
	case MsgUpdatedList:
		log.Info("UpdateList()", zap.Any("resp", msg.Ok), zap.Error(msg.Err))
		if msg.Err != nil {
			return m, tea.Quit
		}

		m.status = statusIdle
		msg.List.ListStatus.NumEpisodesWatched = msg.Ok.NumEpisodesWatched
		if msg.Ok.Status != "" {
			msg.List.ListStatus.Status = msg.Ok.Status
		}
		return m, nil
	default:
		if m.status == statusLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141"))
	watchedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("45")).
		Width(3).
		Align(lipgloss.Right)
	totalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Width(3).
		Align(lipgloss.Right)

	var buf strings.Builder
	// #343433 // nice grey
	// #0044b3
	statusLineText := lipgloss.NewStyle().
		Background(lipgloss.Color("#343433")).
		Width(15).
		Bold(true).
		Align(lipgloss.Center)

	if m.status == statusLoading {
		fmt.Fprintf(&buf, "%s %s\n",
			statusLineText.Render(m.status.String()),
			m.spinner.View(),
		)
	} else {
		fmt.Fprintf(&buf, "%s\n", statusLineText.Render(m.status.String()))
	}

	if m.list == nil {
		return buf.String()
	}

	for i, anime := range m.list.Data {
		if i == m.selected {
			buf.WriteString("👉 ")
		} else {
			buf.WriteString("   ")
		}
		fmt.Fprintf(&buf, "%s / %s %s\n",
			watchedStyle.Render(fmt.Sprint(anime.ListStatus.NumEpisodesWatched)),
			totalStyle.Render(fmt.Sprint(anime.Node.NumEpisodes)),
			titleStyle.Render(PrefTitle(&anime.Node)),
		)
	}

	return buf.String()
}

func addNumWatchedEpisodes(n int, m model) func() tea.Msg {
	list := &m.list.Data[m.selected]
	opts := mal.UpdateListOpts()
	episodes := list.Node.NumEpisodes
	watched := list.ListStatus.NumEpisodesWatched

	switch {
	case episodes == watched+n:
		opts.Status(mal.StatusCompleted)
		opts.NumWatchedEpisodes(episodes)
		fallthrough
	case episodes > watched+n && watched+n >= 0:
		opts.NumWatchedEpisodes(watched + n)
		return tea.Batch(func() tea.Msg {
			resp, err := m.client.UpdateAnimeList(list.Node.ID, opts)
			return MsgUpdatedList{Ok: resp, Err: err, List: list}
		}, spinner.Tick)
	default:
		return nil
	}
}

func PrefTitle(anime *mal.Anime) string {
	titles := anime.AlternativeTitles
	if titles.En != "" {
		return titles.En
	}
	return anime.Title
}

var log *zap.Logger

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"mal.log"}
	cfg.ErrorOutputPaths = []string{"internal-errors.log"}

	// logger, err := zap.NewDevelopment()
	logger, err := cfg.Build()
	if err != nil {
		zap.L().Fatal("couldn't create development config")
	}
	// logger = zap.NewNop()
	log = logger
	mal.SetLogger(logger)

	err = godotenv.Load()
	if err != nil {
		log.Fatal("couldn't load .env file", zap.Error(err))
	}

	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		log.Fatal("Alas, there's been an error", zap.Error(err))
	}
}

func setupClient() (*mal.Client, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	tokenManager := TokenManager{persistPath: filepath.Join(configDir, "malcli", "token")}

	if err := tokenManager.Auth(); err != nil {
		log.Error("auth failed", zap.Error(err))

		token, err := doOAuth()
		if err != nil {
			return nil, err
		}

		err = tokenManager.SetToken(token)
		if err != nil {
			return nil, err
		}
	}

	client, err := mal.NewClient(tokenManager.token)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func doOAuth() (*oauth2.Token, error) {
	auth := mal.NewOauth(os.Getenv("MAL_CLIENT_ID"))
	challenge := auth.NewCodeVerifier()
	url := auth.AuthCodeURL("myState", challenge)
	codeChan := make(chan string, 1)
	server := http.Server{
		Addr:         "0.0.0.0:8089",
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}
	server.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		log.Info("New request")
		r.ParseForm()
		codeChan <- r.FormValue("code")
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()

		rw.WriteHeader(200)
		go func() {
			_ = server.Shutdown(context.Background())
		}()
	})

	fmt.Printf("Navigate to %s\n", url)
	fmt.Println("Waiting for callback...")
	_ = server.ListenAndServe()

	code := <-codeChan
	close(codeChan)

	fmt.Printf("Received code = %s\n", code)

	token, err := auth.Exchange(context.Background(), code, challenge)
	if err != nil {
		return nil, err
	}

	return token, err
}
