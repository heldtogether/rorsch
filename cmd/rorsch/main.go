package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/heldtogether/rorsch/internal"
)

type model struct {
	commands []*internal.Command
	cursor   int
	viewport viewport.Model
	spinner  spinner.Model
	width    int
	height   int
}

var Version = "dev"

func main() {
	var configPath string
	var showVersion bool
	var logLevel string

	flag.StringVar(&configPath, "c", "rorsch.yml", "Path to config file")
	flag.BoolVar(&showVersion, "version", false, "Print the version and exit")
	flag.StringVar(&logLevel, "log", "", "Set log level (debug, info, warn, error)")
	flag.Parse()

	if showVersion {
		fmt.Printf("rorsch: %s\n", Version)
		os.Exit(0)
	}

	logFile := configureLogging(logLevel, configPath)
	defer logFile.Close()

	config := internal.LoadConfig(configPath)
	m := initialModel(config)

	p := tea.NewProgram(m, tea.WithAltScreen())

	for _, command := range m.commands {
		onCmdOutput := func(c *internal.Command, line string, err error, done bool) {
			slog.Info(fmt.Sprintf("received callback from %s", c.Name), "output", line, "done", done, "error", err)
			p.Send(internal.CommandStreamMsg{
				Command: c,
				Line:    line,
				Done:    done,
				Err:     err,
			})
		}

		onFileUpdate := func(c *internal.Command, message string) {
			p.Send(internal.FileEventMsg{
				Command: c,
				Message: message,
			})
			e := internal.NewExecer(c, onCmdOutput)
			go e.Start()
		}

		w := internal.NewCommandWatcher(command, onFileUpdate)
		go w.Start()

		e := internal.NewExecer(command, onCmdOutput)
		go e.Start()

		defer e.Stop()
	}

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func initialModel(commands []*internal.Command) model {
	s := spinner.New()
	s.Spinner = spinner.Points

	return model{
		commands: commands,
		cursor:   0,
		spinner:  s,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		slog.Debug(fmt.Sprintf("Key pressed: %q\n", msg.String()))
		switch msg.String() {
		case "ctrl+c", "q":
			slog.Debug("Quitting…")
			return m, tea.Quit
		case "up":
			slog.Debug("Arrow up -> move cursor")
			if m.cursor > 0 {
				m.cursor--
				m.viewport.SetContent(m.commands[m.cursor].LogTail)
			}
			// IMPORTANT: don't forward this msg to viewport
			return m, tea.Batch(cmds...)
		case "down":
			slog.Debug("Arrow down -> move cursor")
			if m.cursor < len(m.commands)-1 {
				m.cursor++
				m.viewport.SetContent(m.commands[m.cursor].LogTail)
			}
			// IMPORTANT: don't forward this msg to viewport
			return m, tea.Batch(cmds...)
		default:
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		availableHeight := m.height - layoutOverhead(&m)
		if availableHeight < 5 {
			availableHeight = 5
		}

		old := m.viewport
		m.viewport = viewport.New(m.width-2, availableHeight)
		m.viewport.Style = lipgloss.NewStyle().
			MarginLeft(1).
			Border(lipgloss.NormalBorder())

		// Bind vim-style keys
		km := m.viewport.KeyMap
		km.Up = key.NewBinding(key.WithKeys("k"))
		km.Down = key.NewBinding(key.WithKeys("j"))
		km.Left = key.NewBinding(key.WithKeys("h", "left"))
		km.Right = key.NewBinding(key.WithKeys("l", "right"))
		m.viewport.KeyMap = km

		m.viewport.SetContent(m.commands[m.cursor].LogTail)
		m.viewport.SetYOffset(old.YOffset)
		m.viewport.SetHorizontalStep(3)
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case internal.CommandStreamMsg:
		for _, c := range m.commands {
			if c == msg.Command {
				if msg.Err != nil {
					c.Status = internal.StatusFailed
				} else if msg.Done {
					c.Status = internal.StatusOk
				} else {
					c.Status = internal.StatusTrying
					c.LogTail += msg.Line + "\n"
				}
				if m.commands[m.cursor] == c {
					m.viewport.SetContent(c.LogTail)
				}
			}
		}
	case internal.FileEventMsg:
		for _, c := range m.commands {
			if c == msg.Command {
				c.Status = internal.StatusTrying
				c.LogTail = ""
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	s := "\n🔎 Rorsch\n\n"
	s += RenderTable(&m)
	s += "\n" + m.viewport.View() + "\n"
	s += RenderMenu(&m)
	return s
}

func configureLogging(logLevel string, configPath string) *os.File {
	var level *slog.Level
	switch logLevel {
	case "debug":
		lvl := slog.LevelDebug
		level = &lvl
	case "info":
		lvl := slog.LevelInfo
		level = &lvl
	case "warn":
		lvl := slog.LevelWarn
		level = &lvl
	case "error":
		lvl := slog.LevelError
		level = &lvl
	default:
		// logging not requested
	}

	if level == nil {
		slog.SetDefault(slog.New(internal.DiscardHandler{}))
		return nil
	}

	// Set up slog
	logFile, err := os.OpenFile(".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: *level,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Starting rorsch", "version", Version)
	slog.Debug("Using config file", "path", configPath)

	return logFile
}
