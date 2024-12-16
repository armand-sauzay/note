package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type Note struct {
	path, title, content string
	isDir                bool
	depth                int
	expanded             bool // Track if folder is expanded
}

type Model struct {
	config        *Config
	notes         []Note
	cursor        int
	viewport      viewport.Model
	showSidebar   bool
	width, height int
	styles        Styles
	textInput     textinput.Model
	renaming      bool
	mdRenderer    *glamour.TermRenderer
}

var version = "dev"

func printVersion() {
	fmt.Printf("note version %s\n", version)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.renaming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				if m.textInput.Value() != "" {
					currentPath := m.notes[m.cursor].path
					newPath := filepath.Join(filepath.Dir(currentPath), m.textInput.Value())
					if err := os.Rename(currentPath, newPath); err == nil {
						m.updateNotes()
						for i, note := range m.notes {
							if note.path == newPath {
								m.cursor = i
								break
							}
						}
					}
				}
				m.renaming = false
				m.textInput.Blur()
				return m, nil
			case tea.KeyEsc:
				m.renaming = false
				m.textInput.Blur()
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		heights := m.config.CalculateHeights(msg.Height)
		paddingH := m.config.Layout.Padding.Horizontal

		if m.showSidebar {
			sidebarWidth := m.config.Layout.SidebarWidth + (paddingH * 2)
			viewportWidth := m.width - sidebarWidth - (paddingH * 4)
			m.viewport = viewport.New(viewportWidth, heights.Content)
			if m.mdRenderer != nil {
				m.mdRenderer, _ = glamour.NewTermRenderer(
					glamour.WithAutoStyle(),
					glamour.WithWordWrap(viewportWidth-4),
				)
			}
		} else {
			viewportWidth := m.width - (paddingH * 2)
			m.viewport = viewport.New(viewportWidth, heights.Content)
			if m.mdRenderer != nil {
				m.mdRenderer, _ = glamour.NewTermRenderer(
					glamour.WithAutoStyle(),
					glamour.WithWordWrap(viewportWidth-4),
				)
			}
		}
		m.viewport.YPosition = heights.Header
		m.viewport.Style = m.styles.viewport
		m.updatePreview()

	case tea.KeyMsg:
		// Normal mode handling
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.showSidebar = !m.showSidebar
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updatePreview()
			}
		case "down", "j":
			if m.cursor < len(m.notes)-1 {
				m.cursor++
				m.updatePreview()
			}
		case "enter":
			if len(m.notes) > 0 {
				current := m.notes[m.cursor]
				if current.isDir {
					// Start renaming the folder
					m.renaming = true
					m.textInput.SetValue(current.title)
					m.textInput.Focus()
					return m, textinput.Blink
				} else {
					// Only open editor for files
					cmd := exec.Command(m.config.GetEditor(), current.path)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					tea.ExitAltScreen()
					cmd.Run()
					tea.EnterAltScreen()
					m.updateNotes()
					m.updatePreview()
					return m, tea.ClearScreen
				}
			}
		case "N":
			currentDir := m.getCurrentDirectory()
			baseName := "New Folder"
			newPath := filepath.Join(currentDir, baseName)

			// Find unique name if folder exists
			counter := 1
			for {
				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					break
				}
				newPath = filepath.Join(currentDir, fmt.Sprintf("%s %d", baseName, counter))
				counter++
			}

			if err := os.MkdirAll(newPath, 0755); err == nil {
				m.updateNotes()
				// Find and select the new folder
				for i, note := range m.notes {
					if note.path == newPath {
						m.cursor = i
						break
					}
				}
			}

			m.renaming = true
			m.textInput.SetValue(filepath.Base(newPath))
			m.textInput.Focus()
			return m, textinput.Blink
		case "n":
			currentDir := m.getCurrentDirectory()

			timestamp := time.Now().Format("2006-01-02-150405")
			filename := filepath.Join(currentDir, fmt.Sprintf("note-%s.md", timestamp))

			content := fmt.Sprintf("# New Note\n\nCreated: %s\n",
				time.Now().Format("2006-01-02 15:04:05"))

			if err := os.WriteFile(filename, []byte(content), 0644); err == nil {
				m.updateNotes()
				// Find and select the new note
				for i, note := range m.notes {
					if note.path == filename {
						m.cursor = i
						m.updatePreview()
						break
					}
				}
			}
		case "right", "l":
			if len(m.notes) > 0 && m.notes[m.cursor].isDir {
				m.notes[m.cursor].expanded = true
				m.updateNotes()
			}
		case "left", "h":
			if len(m.notes) > 0 {
				if m.notes[m.cursor].isDir {
					m.notes[m.cursor].expanded = false
					m.updateNotes()
				} else {
					// Collapse parent folder if in one
					for i := m.cursor - 1; i >= 0; i-- {
						if m.notes[i].isDir && m.notes[i].depth < m.notes[m.cursor].depth {
							m.notes[i].expanded = false
							m.cursor = i
							m.updateNotes()
							break
						}
					}
				}
			}
		case "backspace":
			if !m.renaming && len(m.notes) > 0 {
				note := m.notes[m.cursor]

				// Create archive directory if it doesn't exist
				if err := os.MkdirAll(m.config.ArchiveDir, 0755); err == nil {
					// Generate unique name to avoid conflicts
					baseName := filepath.Base(note.path)
					timestamp := time.Now().Format("2006-01-02-150405")
					archiveName := fmt.Sprintf("%s-%s", timestamp, baseName)
					archivePath := filepath.Join(m.config.ArchiveDir, archiveName)

					// Move the file/folder to archive
					if err := os.Rename(note.path, archivePath); err == nil {
						// Update cursor position
						if m.cursor > 0 {
							m.cursor--
						}
						m.updateNotes()
						m.updatePreview()
					}
				}
			}
			return m, nil
		}

		// Add viewport key handling
		if !m.renaming {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var doc strings.Builder
	heights := m.config.CalculateHeights(m.height)
	paddingH := m.config.Layout.Padding.Horizontal

	doc.WriteString(m.styles.RenderHeader(m.width, m.config.DefaultDimensions())("note"))

	if m.renaming {
		inputStyle := m.styles.doc.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.styles.highlight).
			Padding(1, 2).
			MarginTop(m.config.Layout.HeaderGap)

		prompt := "Enter folder name:\n\n" + m.textInput.View()
		doc.WriteString(inputStyle.Render(prompt))
		doc.WriteString("\n")
		doc.WriteString(m.renderFooter())
		return doc.String()
	}

	if len(m.notes) == 0 {
		doc.WriteString(m.styles.doc.Render("No notes found. Press 'n' to create one."))
	} else {
		contentWidth := m.width - (paddingH * 2)
		if m.showSidebar {
			sidebarWidth := m.config.Layout.SidebarWidth + (paddingH * 2)
			contentWidth = m.width - sidebarWidth - (paddingH * 4)

			sidebar := m.styles.RenderSidebar(
				m.config.Layout.SidebarWidth,
				heights.Content,
				m.config.Layout.HeaderGap,
			)(m.formatSidebarContent())

			content := m.renderContent(contentWidth, heights)
			doc.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content))
		} else {
			doc.WriteString(m.renderContent(contentWidth, heights))
		}
	}

	doc.WriteString("\n")
	doc.WriteString(m.renderFooter())
	return doc.String()
}

func (m *Model) updatePreview() {
	if len(m.notes) > 0 && m.cursor < len(m.notes) {
		content, err := os.ReadFile(m.notes[m.cursor].path)
		if err == nil {
			m.notes[m.cursor].content = string(content)
			rendered := m.renderMarkdown(string(content))
			m.viewport.SetContent(rendered)
			m.viewport.GotoTop()
		}
	}
}

func (m *Model) updateNotes() {
	expandedFolders := make(map[string]bool)
	for _, note := range m.notes {
		if note.isDir {
			expandedFolders[note.path] = note.expanded
		}
	}

	var walkNotes func(dir string, depth int) []Note
	walkNotes = func(dir string, depth int) []Note {
		var notes []Note
		files, _ := os.ReadDir(dir)
		for _, f := range files {
			path := filepath.Join(dir, f.Name())

			// Skip archive directory
			if m.isArchiveDir(path) {
				continue
			}

			if f.IsDir() {
				folderNote := Note{
					path:     path,
					title:    f.Name(),
					isDir:    true,
					depth:    depth,
					expanded: expandedFolders[path],
				}
				notes = append(notes, folderNote)
				if folderNote.expanded {
					notes = append(notes, walkNotes(path, depth+1)...)
				}
			} else if strings.HasSuffix(f.Name(), ".md") {
				content, _ := os.ReadFile(path)
				title := extractTitle(string(content))
				if title == "" {
					title = strings.TrimSuffix(f.Name(), ".md")
				}
				notes = append(notes, Note{
					path:    path,
					title:   title,
					content: string(content),
					depth:   depth,
				})
			}
		}
		return notes
	}

	m.notes = walkNotes(".", 0)
}

func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func initialModel() (Model, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return Model{}, err
	}

	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	heights := cfg.CalculateHeights(height)
	paddingH := cfg.Layout.Padding.Horizontal

	vp := viewport.New(width-(paddingH*2), heights.Content)
	vp.YPosition = heights.Header

	ti := textinput.New()
	ti.Placeholder = "Folder name"
	ti.CharLimit = 50
	ti.Width = 30

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-(paddingH*2)-4),
	)
	if err != nil {
		renderer = nil // Will fallback to plain text
	}

	m := Model{
		config:      cfg,
		showSidebar: true,
		viewport:    vp,
		width:       width,
		height:      height,
		styles:      NewStyles(cfg),
		textInput:   ti,
		mdRenderer:  renderer,
	}

	os.MkdirAll(cfg.NotesDir, 0755)
	os.Chdir(cfg.NotesDir)

	m.updateNotes()
	m.updatePreview()
	return m, nil
}

func (m Model) formatSidebarContent() string {
	var sidebarContent strings.Builder

	for i, note := range m.notes {
		style := lipgloss.NewStyle()
		if i == m.cursor {
			style = style.Foreground(m.styles.highlight)
		}

		indent := strings.Repeat("  ", note.depth)
		var icon string

		if note.isDir {
			if note.expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
		} else {
			if i < len(m.notes)-1 && m.notes[i+1].depth >= note.depth {
				icon = "├─ "
			} else {
				icon = "└─ "
			}
		}

		line := indent + icon + note.title
		sidebarContent.WriteString(style.Render(line) + "\n")
	}
	return sidebarContent.String()
}

func (m Model) formatStatusBarContent() string {
	statusText := fmt.Sprintf("%d notes", len(m.notes))
	if m.cursor < len(m.notes) {
		statusText = fmt.Sprintf("%s • %s", m.notes[m.cursor].title, statusText)
	}
	return statusText
}

func (m Model) renderContent(width int, heights struct{ Content, Header, Footer int }) string {
	return m.styles.RenderContent(
		width,
		heights.Content,
		m.config.Layout.HeaderGap,
	)(m.viewport.View())
}

func (m Model) renderFooter() string {
	if m.renaming {
		return m.styles.RenderStatusBar(m.width)("Enter to confirm • Esc to cancel")
	}

	statusText := m.formatStatusBarContent()
	helpText := "↑/k,↓/j: up/down • h/l: expand • enter: edit • n: new note • N: new folder • backspace: archive • tab: show sidebar • q: quit"

	var footer strings.Builder
	footer.WriteString(m.styles.RenderStatusBar(m.width)(statusText))
	footer.WriteString("\n")
	footer.WriteString(m.styles.RenderStatusBar(m.width)(helpText))

	return footer.String()
}

func (m Model) getCurrentDirectory() string {
	if len(m.notes) == 0 {
		return "."
	}

	current := m.notes[m.cursor]
	if current.isDir {
		return current.path
	}

	// If it's a file, find its parent directory
	for i := m.cursor; i >= 0; i-- {
		if m.notes[i].isDir && m.notes[i].depth < current.depth {
			return m.notes[i].path
		}
	}
	return "."
}

func (m Model) isArchiveDir(path string) bool {
	return path == m.config.ArchiveDir || filepath.Base(path) == "archive"
}

func (m *Model) renderMarkdown(content string) string {
	if m.mdRenderer == nil {
		return content
	}

	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return content
	}

	return rendered
}

func main() {
	// Add config flag alongside version flag
	versionFlag := flag.Bool("version", false, "Print version information")
	configFlag := flag.Bool("config", false, "Print configuration file location and contents")
	flag.Parse()

	// Check if version flag was provided
	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	// Check if config flag was provided
	if *configFlag {
		printConfig()
		os.Exit(0)
	}

	model, err := initialModel()
	if err != nil {
		log.Fatalf("Failed to initialize model: %v", err)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func printConfig() {
	configDir, err := getConfigDir()

	configPath := filepath.Join(configDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
