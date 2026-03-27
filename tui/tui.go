package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/manager"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	manager   manager.Manager
	power     bool
	locs      map[api.ObjectAddress]*api.Loc
	addresses []api.ObjectAddress
	cursor    int
}

type powerMsg api.Power
type locMsg api.Loc

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "p":
			m.power = !m.power
			m.manager.SetPowerRequest(api.PowerState{Enabled: m.power})
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
					if req.Speed < 100 {
						req.Speed += 10
						if req.Speed > 100 {
							req.Speed = 100
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
					if req.Speed > 0 {
						if req.Speed >= 10 {
							req.Speed -= 10
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
	}

	return m, nil
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

func (m *model) View() string {
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
		speed := 0
		dir := "FORWARD"
		if loc.GetRequest() != nil {
			speed = int(loc.GetRequest().GetSpeed())
			if loc.GetRequest().GetDirection() == api.LocDirection_REVERSE {
				dir = "REVERSE"
			}
		}

		s.WriteString(fmt.Sprintf("%s %s: Speed %3d%%, Dir %s\n", cursor, style.Render(string(addr)), speed, dir))
	}

	s.WriteString("\nControls: +/- Speed, [/] Direction, p Power, q Quit\n")

	return s.String()
}

func Run(ctx context.Context, mgr manager.Manager) error {
	m := &model{
		manager: mgr,
		locs:    make(map[api.ObjectAddress]*api.Loc),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start goroutines to feed messages to Bubble Tea
	pCh, pCancel := mgr.SubscribePowerActuals(true, time.Second)
	defer pCancel()
	lCh, lCancel := mgr.SubscribeLocActuals(true, time.Second)
	defer lCancel()

	go func() {
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

	_, err := p.Run()
	return err
}
