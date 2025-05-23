package main

import (
	"flag"
	"fmt"
	"os"

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

func main() {

	var configPath string
	flag.StringVar(&configPath, "c", "rorsch.yml", "Path to config file")
	flag.Parse()

	config := internal.LoadConfig(configPath)
	m := initialModel(config)

	p := tea.NewProgram(m, tea.WithAltScreen())

	for _, command := range m.commands {
		onCmdOutput := func(c *internal.Command, line string, err error, done bool) {
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
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.commands)-1 {
				m.cursor++
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(msg.Width-4, msg.Height/2)
		m.viewport.SetContent(m.commands[m.cursor].LogTail)
		m.viewport.Style = lipgloss.NewStyle().
			MarginLeft(2).
			Border(lipgloss.NormalBorder())
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
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
	return m, cmd
}

func (m model) View() string {
	s := "\n🔎 Rorsch\n\n"

	s += RenderTable(&m)

	availableHeight := m.height - layoutOverhead(&m)
	if availableHeight < 5 {
		availableHeight = 5 // don't let viewport collapse
	}

	m.viewport = viewport.New(m.width-2, availableHeight)
	m.viewport.SetContent(m.commands[m.cursor].LogTail)
	m.viewport.Style = lipgloss.NewStyle().MarginLeft(1).Border(lipgloss.NormalBorder())

	s += "\n" + m.viewport.View() + "\n"

	s += RenderMenu(&m)

	return s
}
