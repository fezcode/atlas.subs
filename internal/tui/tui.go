package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fezcode/atlas.subs/internal/api"
)

type state int

const (
	stateSearch state = iota
	stateLoading
	stateList
	stateActionMenu // "Download" or "View"
	stateDownloading
	stateDone
	stateView
)

// ── Custom List Delegate ─────────────────────────────────────────

type customDelegate struct{}

func (d customDelegate) Height() int                               { return 2 }
func (d customDelegate) Spacing() int                              { return 0 }
func (d customDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd   { return nil }
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	title := i.subtitle.SubFileName
	desc := fmt.Sprintf("Lang: %s | Release: %s", i.subtitle.LanguageName, i.subtitle.MovieReleaseName)

	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("▶ "+title)+"\n"+itemDescStyle.Copy().Foreground(colorPrimary).Render("  "+desc))
	} else {
		fmt.Fprint(w, normalItemStyle.Render(title)+"\n"+itemDescStyle.Render(desc))
	}
}

// ── Model ────────────────────────────────────────────────────────

type item struct {
	subtitle api.Subtitle
}

func (i item) Title() string       { return i.subtitle.SubFileName }
func (i item) Description() string { return fmt.Sprintf("Language: %s | Release: %s", i.subtitle.LanguageName, i.subtitle.MovieReleaseName) }
func (i item) FilterValue() string { return i.subtitle.SubFileName + " " + i.subtitle.LanguageName }

type model struct {
	state           state
	textInput       textinput.Model
	list            list.Model
	spinner         spinner.Model
	viewport        viewport.Model
	err             error
	downloadPath    string
	viewContent     string
	width           int
	height          int
	selectedSub     api.Subtitle
	actionIndex     int
	actionOptions   []string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Matrix Revolutions..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorPrimary)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	delegate := customDelegate{}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	vp := viewport.New(0, 0)

	return model{
		state:         stateSearch,
		textInput:     ti,
		list:          l,
		spinner:       s,
		viewport:      vp,
		actionOptions: []string{"Download", "View"},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type searchResultMsg []api.Subtitle
type errMsg struct{ err error }
type downloadDoneMsg string
type viewDoneMsg string

func searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		subs, err := api.Search(query)
		if err != nil {
			return errMsg{err}
		}
		return searchResultMsg(subs)
	}
}

func downloadCmd(sub api.Subtitle) tea.Cmd {
	return func() tea.Msg {
		path, err := api.DownloadSubtitle(sub)
		if err != nil {
			return errMsg{err}
		}
		return downloadDoneMsg(path)
	}
}

func viewCmd(sub api.Subtitle) tea.Cmd {
	return func() tea.Msg {
		path, err := api.DownloadSubtitle(sub)
		if err != nil {
			return errMsg{err}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return errMsg{err}
		}
		// Try to delete the file since it's just for viewing, but ignore errors
		_ = os.Remove(path)

		return viewDoneMsg(string(content))
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.width = msg.Width - h
		m.height = msg.Height - v
		m.list.SetSize(m.width, m.height-6) // Adjust for banner and padding
		m.viewport.Width = m.width
		m.viewport.Height = m.height - 6

	case searchResultMsg:
		m.state = stateList
		var items []list.Item
		for _, s := range msg {
			items = append(items, item{subtitle: s})
		}
		m.list.SetItems(items)
		return m, nil

	case downloadDoneMsg:
		m.state = stateDone
		m.downloadPath = string(msg)
		return m, tea.Quit

	case viewDoneMsg:
		m.state = stateView
		m.viewContent = string(msg)
		m.viewport.SetContent(m.viewContent)
		m.viewport.GotoTop()
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateDone
		return m, tea.Quit
	}

	switch m.state {
	case stateSearch:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				val := strings.TrimSpace(m.textInput.Value())
				if val != "" {
					m.state = stateLoading
					return m, tea.Batch(m.spinner.Tick, searchCmd(val))
				}
			} else if msg.Type == tea.KeyEsc {
				return m, tea.Quit
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case stateLoading, stateDownloading:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case stateList:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				selected := m.list.SelectedItem()
				if selected != nil {
					m.selectedSub = selected.(item).subtitle
					m.state = stateActionMenu
					m.actionIndex = 0
					return m, nil
				}
			} else if msg.Type == tea.KeyEsc {
				m.state = stateSearch
				m.textInput.Focus()
				return m, nil
			}
		}
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case stateActionMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				m.actionIndex--
				if m.actionIndex < 0 {
					m.actionIndex = len(m.actionOptions) - 1
				}
			case "down", "j", "tab":
				m.actionIndex++
				if m.actionIndex >= len(m.actionOptions) {
					m.actionIndex = 0
				}
			case "enter":
				if m.actionIndex == 0 {
					// Download
					m.state = stateDownloading
					return m, tea.Batch(m.spinner.Tick, downloadCmd(m.selectedSub))
				} else {
					// View
					m.state = stateDownloading
					return m, tea.Batch(m.spinner.Tick, viewCmd(m.selectedSub))
				}
			case "esc":
				m.state = stateList
				return m, nil
			}
		}

	case stateView:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc || msg.String() == "q" {
				m.state = stateList
				return m, nil
			}
		}
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return appStyle.Render(fmt.Sprintf("%s\n\n%s", banner(), dangerStyle.Render(fmt.Sprintf("Error: %v", m.err))))
	}

	var content string

	switch m.state {
	case stateSearch:
		content = fmt.Sprintf(
			"%s\n%s\n\n%s\n\n%s",
			banner(),
			titleStyle.Render("Search Subtitles"),
			m.textInput.View(),
			helpStyle.Render("Press Enter to search • Esc to quit"),
		)

	case stateLoading:
		content = fmt.Sprintf(
			"%s\n%s\n\n%s %s",
			banner(),
			titleStyle.Render("Searching..."),
			m.spinner.View(),
			mutedStyle.Render("Connecting to OpenSubtitles..."),
		)

	case stateList:
		content = fmt.Sprintf(
			"%s\n%s\n\n%s",
			banner(),
			titleStyle.Render("Search Results"),
			m.list.View(),
		)

	case stateActionMenu:
		menu := strings.Builder{}
		menu.WriteString(fmt.Sprintf("%s\n%s\n\n", banner(), titleStyle.Render("Select Action")))
		menu.WriteString(subtitleStyle.Render(m.selectedSub.SubFileName) + "\n\n")

		for i, opt := range m.actionOptions {
			cursor := "  "
			style := normalItemStyle
			if i == m.actionIndex {
				cursor = lipgloss.NewStyle().Foreground(colorPrimary).Render("▶ ")
				style = selectedItemStyle
			}
			menu.WriteString(cursor + style.Render(opt) + "\n")
		}
		menu.WriteString("\n" + helpStyle.Render("Use j/k to move • Enter to select • Esc to cancel"))
		content = menu.String()

	case stateDownloading:
		content = fmt.Sprintf(
			"%s\n%s\n\n%s %s",
			banner(),
			titleStyle.Render("Processing..."),
			m.spinner.View(),
			mutedStyle.Render(fmt.Sprintf("Downloading %s...", m.selectedSub.SubFileName)),
		)

	case stateView:
		header := titleStyle.Render("View Subtitle") + " " + subtitleStyle.Render(m.selectedSub.SubFileName)
		footer := helpStyle.Render(fmt.Sprintf(" %3.f%% • Use ↑/↓ to scroll • Esc/q to back", m.viewport.ScrollPercent()*100))
		content = fmt.Sprintf("%s\n%s\n%s\n%s", banner(), header, m.viewport.View(), footer)

	case stateDone:
		content = fmt.Sprintf(
			"%s\n%s\n\n%s\n%s\n\n%s",
			banner(),
			titleStyle.Render("Success!"),
			successStyle.Render("✓ Subtitle downloaded and extracted."),
			valueStyle.Render(m.downloadPath),
			helpStyle.Render("Press any key or Ctrl+C to exit."),
		)
	}

	return appStyle.Render(content)
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return err
	}
	
	if finalModel, ok := m.(model); ok {
		if finalModel.err != nil {
			return finalModel.err
		}
	}
	return nil
}
