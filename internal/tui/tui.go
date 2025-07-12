package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Richd0tcom/schedrift/internal/models"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#0D3B66")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#F25F5C")).
			Padding(0, 1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1E88E5"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#F9DC4B")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#2A9D8F")).
			Padding(0, 1)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF9800"))
)

// Model represents the TUI model
type Model struct {
	spinner           spinner.Model
	viewport          viewport.Model
	content           string
	loading           bool
	loadingMsg        string
	error             error
	schema            *models.DatabaseSchema
	width             int
	height            int
	ready             bool
	schemaFetching    bool //might be same as loading
	connectionMessage string
}

type SchemaFetchedMsg struct {
	Schema *models.DatabaseSchema
}

type ConnectionMsg struct {
	Message string
}

// NewModel creates a new TUI model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(100, 30)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#90CAF9")).
		PaddingRight(1)

	return Model{
		spinner:    s,
		viewport:   vp,
		loading:    false,
		loadingMsg: "",
		content:    "",
		error:      nil,
		schema:     nil,
		width:      80,
		height:     24,
		ready:      false,
	}
}

// Init initializes the TUI model
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// StartLoading starts loading with a message
func (m *Model) StartLoading(msg string) {
	m.loading = true
	m.loadingMsg = msg
}

// StopLoading stops loading
func (m *Model) StopLoading() {
	m.loading = false
	m.loadingMsg = ""
}

// SetContent sets the content to display
func (m *Model) SetContent(content string) {
	m.content = content
}

// SetError sets an error
func (m *Model) SetError(err error) {
	m.error = err
}

// SetSchema sets the schema
func (m *Model) SetSchema(schema *models.DatabaseSchema) {
	m.schema = schema
}

// LoadingMsg is a command to set loading message
type LoadingMsg string

// ContentMsg is a command to set content
type ContentMsg string

// ErrorMsg is a command to set error
type ErrorMsg struct {
	Err error
}

// SchemaMsg is a command to set schema
type SchemaMsg struct {
	Schema *models.DatabaseSchema
}

func StartExtraction(extract func() (*models.DatabaseSchema, error)) tea.Cmd {
	return func() tea.Msg {
		schema, err := extract()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return SchemaFetchedMsg{Schema: schema}
	}
}

// Update updates the model based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := 2
		footerHeight := 1

		if !m.ready {
			// First time initialization
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			// Subsequent resizes
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}

		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		return m, spinnerCmd

	case LoadingMsg:
		m.loading = true
		m.loadingMsg = string(msg)
		return m, m.spinner.Tick

	case ContentMsg:
		m.content = string(msg)
		m.viewport.SetContent(string(msg))
		return m, nil

	case ErrorMsg:
		m.error = msg.Err
		m.loading = false
		return m, nil

	case SchemaMsg:
		m.schema = msg.Schema
		m.loading = false
		if m.schema != nil {
			m.content = buildSchemaView(m.schema)
			m.viewport.SetContent(m.content)
		}
		return m, nil

	case SchemaFetchedMsg:
		m.schemaFetching = false
		m.schema = msg.Schema
		m.content = buildSchemaView(m.schema)
		m.viewport.SetContent(m.content)
		return m, nil

	case ConnectionMsg:
		m.connectionMessage = msg.Message
		return m, nil
	}

	// Handle viewport updates
	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, cmd
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var connInfo string

	// Header
	header := titleStyle.Render("Schema Drift Detector")
	header = lipgloss.JoinHorizontal(lipgloss.Top, header)
	header = lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(header)

	// Connection info
	if m.connectionMessage != "" {
		connInfo = infoStyle.Render(m.connectionMessage)
	}

	// Content
	content := m.viewport.View()

	// Loading indicator or footer
	var footer string
	if m.loading {
		loadingText := fmt.Sprintf("%s %s", m.spinner.View(), m.loadingMsg)
		footer = infoStyle.Render(loadingText)
	} else if m.error != nil {
		footer = errorStyle.Render(fmt.Sprintf("Error: %v", m.error))
	} else {
		footer = infoStyle.Render("Press q to quit")
	}

	// Combine everything
	return fmt.Sprintf("%s\n%s\n%s\n%s", header, connInfo, content, footer)
}

// buildSchemaView creates a text representation of the schema
func buildSchemaView(dbSchema *models.DatabaseSchema) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Database: %s\n\n", dbSchema.Name))

	for _, schema := range dbSchema.Schemas {
		builder.WriteString(fmt.Sprintf("Schema: %s\n\n", schema.Name))

		// Tables
		builder.WriteString(fmt.Sprintf("Tables (%d):\n", len(schema.Tables)))
		builder.WriteString("----------------------------------------\n")

		// Get sorted table names
		tableNames := make([]string, 0, len(schema.Tables))
		for _, table := range schema.Tables {
			tableNames = append(tableNames, table.Name)
		}
		sort.Strings(tableNames)

		for i := range tableNames {
			table := schema.Tables[i]
			builder.WriteString(fmt.Sprintf("  • %s\n", table.Name))

			// Columns
			for _, col := range table.Columns {
				nullable := "NOT NULL"
				if col.IsNullable {
					nullable = "NULL"
				}
				builder.WriteString(fmt.Sprintf("    - %s: %s %s\n", col.Name, col.DataType, nullable))
			}

			// Constraints
			if len(table.Constraints) > 0 {
				builder.WriteString("    Constraints:\n")
				for _, constraint := range table.Constraints {
					builder.WriteString(fmt.Sprintf("    - %s (%s)\n", constraint.Name, constraint.Type))
				}
			}

			// Indexes
			if len(schema.Indexes) > 0 {
				builder.WriteString("    Indexes:\n")
				for _, index := range schema.Indexes {
					unique := ""
					if index.IsUnique {
						unique = "UNIQUE "
					}
					builder.WriteString(fmt.Sprintf("    - %s%s (%s)\n", unique, index.Name, strings.Join(index.Columns, ", ")))
				}
			}
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// RunTUI runs the TUI with the given schema
func RunTUI(schema *models.DatabaseSchema) error {
	m := NewModel()
	m.schema = schema

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// SetConnectionInfo sets the connection information
func SetConnectionInfo(message string) tea.Cmd {
	return func() tea.Msg {
		return ConnectionMsg{Message: message}
	}
}

func init() {
	// Properly initialize any needed values
}
