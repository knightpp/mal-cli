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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/knightpp/mal-api/mal"
	"github.com/knightpp/mal-cli/widget"
	"github.com/knightpp/mal-cli/widget/initialize"
	"github.com/knightpp/mal-cli/widget/recent"
	"github.com/knightpp/mal-cli/widget/status"
)

type model struct {
	// widgets
	widgetStack widget.WidgetStack

	client *mal.Client
}

func initialModel() model {
	sl := status.NewStatusLine(initialize.New())
	return model{
		widgetStack: widget.NewWidgetStack(sl),
	}
}

func initClient() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		client, err := setupClient()
		if err != nil {
			return tea.Quit
		}
		return MsgClientReady{client}
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.widgetStack.Init(), initClient())
}

type MsgClientReady struct {
	client *mal.Client
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MsgClientReady:
		m.client = msg.client
		m.widgetStack.Pop()
		newWidget := status.NewStatusLine(recent.New(m.client))
		newWidget.SetStatus(status.Idle)
		m.widgetStack.Push(newWidget)
		return m, newWidget.Init()
	default:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
		return m, m.widgetStack.Update(msg)
	}

}

// type batch struct {
// 	buf []tea.Cmd
// }

// func (b *batch) Add(cmd tea.Cmd) {
// 	if cmd != nil {
// 		b.buf = append(b.buf, cmd)
// 	}
// }
// func (b *batch) Cmd() tea.Cmd {
// 	if len(b.buf) == 0 {
// 		return nil
// 	} else if len(b.buf) == 1 {
// 		return b.buf[0]
// 	}
// 	return tea.Batch(b.buf...)
// }

func (m model) View() string {
	var buf strings.Builder
	m.widgetStack.Peek().View(&buf)
	return buf.String()
}

var log *zap.Logger

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"mal.log"}
	cfg.ErrorOutputPaths = []string{"mal.log", "stderr"}

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
		log.Warn("couldn't load .env file", zap.Error(err))
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
	clientID := os.Getenv("MAL_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("environmental variable MAL_CLIENT_ID is empty or unset")
	}

	auth := mal.NewOauth(clientID)
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
