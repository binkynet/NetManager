package tui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/manager"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	manager   manager.Manager
	power     bool
	locs      map[api.ObjectAddress]*api.Loc
	addresses []api.ObjectAddress
	cursor    int
	viewport  viewport.Model
	logs      []string
	ready     bool

	// Popup state
	showAddLocPopup  bool
	textInput        textinput.Model
	activeButton     int // 0 for Ok, 1 for Cancel
	speedStepsCursor int // 0: 14, 1: 28, 2: 128
	popupFocus       int // 0: Address, 1: SpeedSteps, 2: Buttons
}

var speedStepsOptions = []int32{14, 28, 128}

type powerMsg api.Power
type locMsg api.Loc
type logMsg string

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.showAddLocPopup {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.showAddLocPopup = false
				return m, nil
			case "tab", "shift+tab":
				if msg.String() == "tab" {
					m.popupFocus = (m.popupFocus + 1) % 3
				} else {
					m.popupFocus = (m.popupFocus + 2) % 3
				}
				if m.popupFocus == 0 {
					m.textInput.Focus()
				} else {
					m.textInput.Blur()
				}
				return m, nil
			case "left", "right":
				if m.popupFocus == 1 {
					if msg.String() == "left" {
						m.speedStepsCursor = (m.speedStepsCursor + 2) % 3
					} else {
						m.speedStepsCursor = (m.speedStepsCursor + 1) % 3
					}
				} else if m.popupFocus == 2 {
					m.activeButton = (m.activeButton + 1) % 2
				}
			case "up", "down":
				if msg.String() == "up" {
					m.popupFocus = (m.popupFocus + 2) % 3
				} else {
					m.popupFocus = (m.popupFocus + 1) % 3
				}
				if m.popupFocus == 0 {
					m.textInput.Focus()
				} else {
					m.textInput.Blur()
				}
				return m, nil
			case "enter":
				if m.popupFocus == 2 && m.activeButton == 1 {
					// Cancel
					m.showAddLocPopup = false
				} else {
					// Ok (or enter in input/speedsteps)
					addr := api.ObjectAddress(m.textInput.Value())
					if addr != "" {
						if _, ok := m.locs[addr]; !ok {
							m.locs[addr] = &api.Loc{
								Address: addr,
								Request: &api.LocState{
									SpeedSteps: speedStepsOptions[m.speedStepsCursor],
								},
							}
							m.updateAddresses()
						}
					}
					m.showAddLocPopup = false
				}
			}
		}

		if m.popupFocus == 0 {
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "p":
			m.power = !m.power
			m.manager.SetPowerRequest(api.PowerState{Enabled: m.power})
		case "a":
			m.showAddLocPopup = true
			m.textInput = textinput.New()
			m.textInput.Placeholder = "Loc Address"
			m.textInput.Focus()
			m.textInput.CharLimit = 64
			m.textInput.Width = 20
			m.activeButton = 0
			m.speedStepsCursor = 2 // Default 128
			m.popupFocus = 0
			return m, nil
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.addresses)-1 {
				m.cursor++
			}
		case "+":
			if len(m.addresses) > 0 {
				addr := m.addresses[m.cursor]
				if loc, ok := m.locs[addr]; ok {
					req := loc.GetRequest().Clone()
					if req == nil {
						req = &api.LocState{SpeedSteps: 128}
					}
					maxSteps := req.GetSpeedSteps()
					if maxSteps <= 0 {
						maxSteps = 128
					}
					step := maxSteps / 10
					if step < 1 {
						step = 1
					}
					if req.Speed < maxSteps {
						req.Speed += step
						if req.Speed > maxSteps {
							req.Speed = maxSteps
						}
						m.manager.SetLocRequest(api.Loc{
							Address: addr,
							Request: req,
						})
					}
				}
			}
		case "-":
			if len(m.addresses) > 0 {
				addr := m.addresses[m.cursor]
				if loc, ok := m.locs[addr]; ok {
					req := loc.GetRequest().Clone()
					if req == nil {
						req = &api.LocState{SpeedSteps: 128}
					}
					maxSteps := req.GetSpeedSteps()
					if maxSteps <= 0 {
						maxSteps = 128
					}
					step := maxSteps / 10
					if step < 1 {
						step = 1
					}
					if req.Speed > 0 {
						if req.Speed >= step {
							req.Speed -= step
						} else {
							req.Speed = 0
						}
						m.manager.SetLocRequest(api.Loc{
							Address: addr,
							Request: req,
						})
					}
				}
			}
		case "[":
			if len(m.addresses) > 0 {
				addr := m.addresses[m.cursor]
				if loc, ok := m.locs[addr]; ok {
					req := loc.GetRequest().Clone()
					if req == nil {
						req = &api.LocState{SpeedSteps: 128}
					}
					req.Direction = api.LocDirection_FORWARD
					m.manager.SetLocRequest(api.Loc{
						Address: addr,
						Request: req,
					})
				}
			}
		case "]":
			if len(m.addresses) > 0 {
				addr := m.addresses[m.cursor]
				if loc, ok := m.locs[addr]; ok {
					req := loc.GetRequest().Clone()
					if req == nil {
						req = &api.LocState{SpeedSteps: 128}
					}
					req.Direction = api.LocDirection_REVERSE
					m.manager.SetLocRequest(api.Loc{
						Address: addr,
						Request: req,
					})
				}
			}
		}

	case powerMsg:
		p := api.Power(msg)
		m.power = p.GetActual().GetEnabled()

	case locMsg:
		l := api.Loc(msg)
		m.locs[l.Address] = &l
		m.updateAddresses()

	case logMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 500 {
			m.logs = m.logs[len(m.logs)-500:]
		}
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		m.viewport.GotoBottom()

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight + 2 // +2 for spacing

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight + 1
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) updateAddresses() {
	m.addresses = make([]api.ObjectAddress, 0, len(m.locs))
	for addr := range m.locs {
		m.addresses = append(m.addresses, addr)
	}
	sort.Slice(m.addresses, func(i, j int) bool {
		return string(m.addresses[i]) < string(m.addresses[j])
	})
}

func (m *model) headerView() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render("BinkyNet Network Manager"))
	s.WriteString("\n\n")

	powerStr := "OFF"
	if m.power {
		powerStr = "ON"
	}
	s.WriteString(fmt.Sprintf("Global Power: %s (press 'p' to toggle)\n\n", powerStr))

	s.WriteString("Trains:\n")
	for i, addr := range m.addresses {
		cursor := " "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = ">"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		}

		loc := m.locs[addr]
		req := loc.GetRequest()
		act := loc.GetActual()

		var stateStr string
		if req != nil && act != nil && !req.Equal(act) {
			stateStr = fmt.Sprintf("Req: %s | Act: %s", formatLocState(req), formatLocState(act))
		} else if req != nil {
			stateStr = formatLocState(req)
		} else if act != nil {
			stateStr = formatLocState(act)
		} else {
			stateStr = "Unknown"
		}

		s.WriteString(fmt.Sprintf("%s %s: %s\n", cursor, style.Render(string(addr)), stateStr))
	}

	s.WriteString("\nControls: +/- Speed, [/] Direction, p Power, a Add Loc, q Quit\n")
	return s.String()
}

func formatLocState(state *api.LocState) string {
	if state == nil {
		return "Unknown"
	}
	speed := int(state.GetSpeed())
	maxSteps := int(state.GetSpeedSteps())
	if maxSteps <= 0 {
		maxSteps = 128
	}
	dir := "FORWARD"
	if state.GetDirection() == api.LocDirection_REVERSE {
		dir = "REVERSE"
	}
	return fmt.Sprintf("Speed %d/%d, Dir %s", speed, maxSteps, dir)
}

func (m *model) footerView() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#3C3C3C")).
		Padding(0, 1).
		Render("Logs")
}

var (
	popupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Background(lipgloss.Color("#202020"))

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#3C3C3C")).
			Padding(0, 3).
			MarginTop(1).
			MarginRight(2)

	activeButtonStyle = buttonStyle.
				Background(lipgloss.Color("#7D56F4"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			MarginRight(1)

	choiceStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginRight(1).
			Background(lipgloss.Color("#3C3C3C"))

	activeChoiceStyle = choiceStyle.
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FAFAFA"))
)

func (m *model) renderPopup() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Add Locomotive"))
	s.WriteString("\n\n")

	// Address input
	addrLabel := labelStyle.Render("Address:")
	if m.popupFocus == 0 {
		addrLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render("Address:")
	}
	s.WriteString(addrLabel)
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")

	// Speed steps combo
	s.WriteString(labelStyle.Render("Speed Steps:"))
	for i, opt := range speedStepsOptions {
		style := choiceStyle
		if i == m.speedStepsCursor {
			if m.popupFocus == 1 {
				style = activeChoiceStyle
			} else {
				style = choiceStyle.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#7D56F4"))
			}
		}
		s.WriteString(style.Render(fmt.Sprintf("%d", opt)))
	}
	s.WriteString("\n\n")

	// Buttons
	okStyle := buttonStyle
	cancelStyle := buttonStyle

	if m.popupFocus == 2 {
		if m.activeButton == 0 {
			okStyle = activeButtonStyle
		} else {
			cancelStyle = activeButtonStyle
		}
	}

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		okStyle.Render("Ok"),
		cancelStyle.Render("Cancel"),
	))

	return popupStyle.Render(s.String())
}

func (m *model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	mainView := fmt.Sprintf("%s\n%s\n%s\n%s",
		m.headerView(),
		m.footerView(),
		m.viewport.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C")).Render("Scroll with Mouse Wheel or PgUp/PgDn"),
	)

	if m.showAddLocPopup {
		// Place popup in the middle (roughly)
		popup := m.renderPopup()
		return lipgloss.Place(
			m.viewport.Width,
			m.viewport.Height+lipgloss.Height(m.headerView())+lipgloss.Height(m.footerView()),
			lipgloss.Center,
			lipgloss.Center,
			popup,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
		)
	}

	return mainView
}

type logWriter struct {
	send func(logMsg)
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for _, line := range lines {
		if line != "" {
			w.send(logMsg(line))
		}
	}
	return len(p), nil
}

func Start(ctx context.Context, mgr manager.Manager, cancel context.CancelFunc) (io.Writer, error) {
	m := &model{
		manager: mgr,
		locs:    make(map[api.ObjectAddress]*api.Loc),
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Start goroutines to feed messages to Bubble Tea
	pCh, pCancel := mgr.SubscribePowerActuals(true, time.Second)
	lCh, lCancel := mgr.SubscribeLocActuals(true, time.Second)

	go func() {
		defer pCancel()
		defer lCancel()
		for {
			select {
			case msg, ok := <-pCh:
				if !ok {
					return
				}
				p.Send(powerMsg(msg))
			case msg, ok := <-lCh:
				if !ok {
					return
				}
				p.Send(locMsg(msg))
			case <-ctx.Done():
				p.Quit()
				return
			}
		}
	}()

	writer := &logWriter{
		send: func(msg logMsg) {
			p.Send(msg)
		},
	}

	go func() {
		defer cancel()
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v", err)
		}
	}()

	return writer, nil
}
