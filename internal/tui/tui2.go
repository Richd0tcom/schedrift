package tui

// import (
// 	"fmt"
// 	"strings"

// 	"github.com/Richd0tcom/schedrift/internal/models"
// 	"github.com/charmbracelet/bubbles/spinner"
// 	"github.com/charmbracelet/bubbles/viewport"
// 	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/charmbracelet/lipgloss"
// )

// var (
// 	titleStyle = lipgloss.NewStyle().
// 		Bold(true).
// 		Foreground(lipgloss.Color("#FFFFFF")).
// 		Background(lipgloss.Color("#0D47A1")).
// 		Padding(0, 1)

// 	infoStyle = lipgloss.NewStyle().
// 		Foreground(lipgloss.Color("#FFFFFF")).
// 		Background(lipgloss.Color("#1976D2")).
// 		Padding(0, 1)

// 	sectionStyle = lipgloss.NewStyle().
// 		Bold(true).
// 		Foreground(lipgloss.Color("#1E88E5"))

// 	successStyle = lipgloss.NewStyle().
// 		Foreground(lipgloss.Color("#4CAF50"))

// 	errorStyle = lipgloss.NewStyle().
// 		Foreground(lipgloss.Color("#F44336"))

// 	highlightStyle = lipgloss.NewStyle().
// 		Foreground(lipgloss.Color("#FF9800"))
// )

// // Model represents the TUI model
// type Model struct {
// 	viewport          viewport.Model
// 	spinner           spinner.Model
// 	schemaFetching    bool
// 	schema            *models.DatabaseSchema
// 	error             error
// 	width             int
// 	height            int
// 	connectionMessage string
// }

// // NewModel creates a new TUI model
// func NewModel() Model {
// 	s := spinner.New()
// 	s.Spinner = spinner.Dot
// 	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#2196F3"))

// 	vp := viewport.New(100, 30)
// 	vp.Style = lipgloss.NewStyle().
// 		BorderStyle(lipgloss.RoundedBorder()).
// 		BorderForeground(lipgloss.Color("#90CAF9")).
// 		PaddingRight(1)

// 	return Model{
// 		spinner:        s,
// 		viewport:       vp,
// 		schemaFetching: true,
// 	}
// }

// // Init initializes the TUI model
// func (m Model) Init() tea.Cmd {
// 	return spinner.Tick
// }

// // Update updates the TUI model based on messages
// func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	var cmds []tea.Cmd

// 	switch msg := msg.(type) {
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "ctrl+c", "q":
// 			return m, tea.Quit
// 		}

// 	case tea.WindowSizeMsg:
// 		m.width = msg.Width
// 		m.height = msg.Height
// 		m.viewport.Width = msg.Width - 4
// 		m.viewport.Height = msg.Height - 6
// 		return m, nil

// 	case SchemaFetchedMsg:
// 		m.schemaFetching = false
// 		m.schema = msg.Schema
// 		m.viewport.SetContent(m.formatSchema())
// 		return m, nil

// 	case ErrorMsg:
// 		m.schemaFetching = false
// 		m.error = msg.Error
// 		m.viewport.SetContent(errorStyle.Render(fmt.Sprintf("Error: %v", msg.Error)))
// 		return m, nil

// 	case ConnectionMsg:
// 		m.connectionMessage = msg.Message
// 		return m, nil
// 	}

// 	if m.schemaFetching {
// 		var cmd tea.Cmd
// 		m.spinner, cmd = m.spinner.Update(msg)
// 		cmds = append(cmds, cmd)
// 	}

// 	var cmd tea.Cmd
// 	m.viewport, cmd = m.viewport.Update(msg)
// 	cmds = append(cmds, cmd)

// 	return m, tea.Batch(cmds...)
// }

// // View renders the TUI
// func (m Model) View() string {
// 	if m.width == 0 {
// 		// Not initialized yet
// 		return "Loading..."
// 	}

// 	sb := strings.Builder{}

// 	// Title
// 	title := titleStyle.Render("Schema Drift Detector")
// 	sb.WriteString(title)
// 	sb.WriteString("\n")

// 	// Connection info
// 	if m.connectionMessage != "" {
// 		connInfo := infoStyle.Render(m.connectionMessage)
// 		sb.WriteString(connInfo)
// 		sb.WriteString("\n")
// 	}

// 	// Spinner or content
// 	if m.schemaFetching {
// 		sb.WriteString(fmt.Sprintf("\n%s Fetching schema...", m.spinner.View()))
// 	} else if m.error != nil {
// 		sb.WriteString(m.viewport.View())
// 	} else {
// 		sb.WriteString(m.viewport.View())
// 	}

// 	sb.WriteString("\n")
// 	sb.WriteString(infoStyle.Render(" Press q to quit "))

// 	return sb.String()
// }

// // formatSchema formats the schema as a string
// func (m Model) formatSchema() string {
// 	if m.schema == nil {
// 		return "No schema loaded"
// 	}

// 	sb := strings.Builder{}

// 	sb.WriteString(sectionStyle.Render(fmt.Sprintf("Database: %s\n\n", m.schema.Name)))

// 	for _, schema := range m.schema.Schemas {
// 		sb.WriteString(sectionStyle.Render(fmt.Sprintf("Schema: %s\n", schema.Name)))

// 		// Tables
// 		sb.WriteString(fmt.Sprintf("\n%s (%d):\n", highlightStyle.Render("Tables"), len(schema.Tables)))
// 		for _, table := range schema.Tables {
// 			sb.WriteString(fmt.Sprintf("  • %s\n", table.Name))
// 			sb.WriteString(fmt.Sprintf("    %d columns, %d constraints\n", len(table.Columns), len(table.Constraints)))
// 		}

// 		// Views
// 		sb.WriteString(fmt.Sprintf("\n%s (%d):\n", highlightStyle.Render("Views"), len(schema.Views)))
// 		for _, view := range schema.Views {
// 			sb.WriteString(fmt.Sprintf("  • %s\n", view.Name))
// 		}

// 		// Functions
// 		sb.WriteString(fmt.Sprintf("\n%s (%d):\n", highlightStyle.Render("Functions"), len(schema.Functions)))
// 		for _, function := range schema.Functions {
// 			sb.WriteString(fmt.Sprintf("  • %s(%s) returns %s\n", function.Name, function.Arguments, function.ReturnType))
// 		}

// 		// Indexes
// 		sb.WriteString(fmt.Sprintf("\n%s (%d):\n", highlightStyle.Render("Indexes"), len(schema.Indexes)))
// 		for _, index := range schema.Indexes {
// 			sb.WriteString(fmt.Sprintf("  • %s on %s (%s)\n", index.Name, index.Table, strings.Join(index.Columns, ", ")))
// 		}

// 		// Triggers
// 		sb.WriteString(fmt.Sprintf("\n%s (%d):\n", highlightStyle.Render("Triggers"), len(schema.Triggers)))
// 		for _, trigger := range schema.Triggers {
// 			sb.WriteString(fmt.Sprintf("  • %s on %s\n", trigger.Name, trigger.Table))
// 		}

// 		sb.WriteString("\n")
// 	}

// 	return sb.String()
// }

// // Messages

// // SchemaFetchedMsg is sent when the schema has been fetched
// type SchemaFetchedMsg struct {
// 	Schema *model.DatabaseSchema
// }

// // ErrorMsg is sent when an error occurs
// type ErrorMsg struct {
// 	Error error
// }

// // ConnectionMsg is sent when connection information is available
// type ConnectionMsg struct {
// 	Message string
// }

// // StartExtraction starts the schema extraction process
// func StartExtraction(extract func() (*model.DatabaseSchema, error)) tea.Cmd {
// 	return func() tea.Msg {
// 		schema, err := extract()
// 		if err != nil {
// 			return ErrorMsg{Error: err}
// 		}
// 		return SchemaFetchedMsg{Schema: schema}
// 	}
// }

// // SetConnectionInfo sets the connection information
// func SetConnectionInfo(message string) tea.Cmd {
// 	return func() tea.Msg {
// 		return ConnectionMsg{Message: message}
// 	}
// }