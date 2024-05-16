package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type keyMap struct {
	Delete key.Binding
	Quit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Delete, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Delete, k.Quit}, // first column
	}
}

var keys = keyMap{
	Delete: key.NewBinding(
		key.WithKeys("d", "D"),
		key.WithHelp("d", "Delete"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
}

type FileInfo struct {
	Path string
	Size int64
}

type model struct {
	table        table.Model
	keys         keyMap
	help         help.Model
	confirming   bool
	selectedFile string
	rows         []table.Row
}

var rootCmd = &cobra.Command{
	Use:   "filebrowser [directory]",
	Short: "File browser CLI",
	Long:  `A CLI application to browse and manage files using Bubble Tea and Cobra.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		var files []FileInfo
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, FileInfo{Path: path, Size: info.Size()})
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking the path: %v\n", err)
			return
		}

		sort.Slice(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})

		rows := make([]table.Row, 0, 10)
		for i, file := range files {
			if i >= 10 {
				break
			}
			dir, filename := filepath.Split(file.Path)
			rows = append(rows, table.Row{
				fmt.Sprintf("%d", i+1),
				filename,
				strings.TrimSuffix(dir, string(os.PathSeparator)),
				HumanReadable(file.Size),
			})
		}

		columns := []table.Column{
			{Title: "#", Width: 5},
			{Title: "Filename", Width: 30},
			{Title: "Directory", Width: 50},
			{Title: "Size", Width: 15},
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		t.SetStyles(s)

		m := model{
			table: t,
			rows:  rows,
			keys:  keys,
			help:  help.New(),
		}

		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}
	},
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y":
				if m.confirming {
					selectedRow := m.table.SelectedRow()
					if selectedRow != nil {
						filename := selectedRow[1]
						dir := selectedRow[2]
						filePath := filepath.Join(dir, filename)
						err := os.Remove(filePath)
						if err == nil {
							m.confirming = false
							// Remove the row from the slice
							m.rows = append(m.rows[:m.table.Cursor()], m.rows[m.table.Cursor()+1:]...)
							// Update the table with the new rows
							m.table.SetRows(m.rows)
							return m, nil
						}
					}
				}
			case "n":
				m.confirming = false
				return m, nil
			}
		}
	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Delete):
				selectedRow := m.table.SelectedRow()
				if selectedRow != nil {
					m.selectedFile = selectedRow[1]
					m.confirming = true
					return m, nil
				}
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.confirming {
		selectedRow := m.table.SelectedRow()
		if selectedRow != nil {
			filename := selectedRow[1]
			return fmt.Sprintf("Are you sure you want to delete %s? (y/n)", filename)
		}
	}
	style := lipgloss.NewStyle().Margin(1, 2, 1, 2)
	helpView := m.help.View(m.keys)
	tableView := style.Render(m.table.View())

	return tableView + "\n" + helpView
}

func HumanReadable(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
