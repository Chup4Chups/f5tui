package ui

import (
	"fmt"
	"strings"

	"f5tui/internal/f5"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type frame struct {
	name    string
	builder func() tview.Primitive
}

type App struct {
	tv        *tview.Application
	pages     *tview.Pages
	client    *f5.Client
	status    *tview.TextView
	input     *tview.InputField
	inputMode string // "" | "cmd" | "filter"
	layout    *tview.Flex
	stack     []frame

	partition string
	filter    string
}

func Run(client *f5.Client, partition string) error {
	a := &App{
		tv:        tview.NewApplication(),
		pages:     tview.NewPages(),
		client:    client,
		partition: partition,
	}

	a.status = tview.NewTextView().SetDynamicColors(true)
	a.input = tview.NewInputField()
	a.input.SetDoneFunc(a.onInputDone)

	a.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.status, 1, 0, false)

	a.tv.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if a.inputMode != "" {
			return ev
		}
		switch ev.Key() {
		case tcell.KeyEscape:
			a.back()
			return nil
		}
		switch ev.Rune() {
		case ':':
			a.openInput("cmd", ":")
			return nil
		case '/':
			a.openInput("filter", "/")
			a.input.SetText(a.filter)
			return nil
		case '?':
			a.push("help", func() tview.Primitive { return helpView() })
			return nil
		}
		return ev
	})

	a.push("virtualservers", func() tview.Primitive { return a.virtualServersView() })
	a.updateStatus("")

	return a.tv.SetRoot(a.layout, true).EnableMouse(true).Run()
}

func (a *App) openInput(mode, label string) {
	a.inputMode = mode
	a.input.SetLabel(label).SetText("")
	a.layout.Clear().
		AddItem(a.pages, 0, 1, true).
		AddItem(a.input, 1, 0, true).
		AddItem(a.status, 1, 0, false)
	a.tv.SetFocus(a.input)
}

func (a *App) closeInput() {
	mode := a.inputMode
	a.inputMode = ""
	a.layout.Clear().
		AddItem(a.pages, 0, 1, true).
		AddItem(a.status, 1, 0, false)
	if len(a.stack) > 0 {
		a.tv.SetFocus(a.pages)
	}
	_ = mode
}

func (a *App) onInputDone(key tcell.Key) {
	text := strings.TrimSpace(a.input.GetText())
	mode := a.inputMode
	a.input.SetText("")
	a.closeInput()
	if key == tcell.KeyEscape {
		if mode == "filter" {
			a.filter = ""
			a.refresh()
		}
		return
	}
	switch mode {
	case "cmd":
		if text != "" {
			a.runCommand(text)
		}
	case "filter":
		a.filter = text
		a.refresh()
	}
}

func (a *App) runCommand(cmd string) {
	parts := strings.Fields(cmd)
	head := parts[0]
	args := parts[1:]
	switch head {
	case "q", "quit", "exit":
		a.tv.Stop()
	case "vs", "virtual", "virtualservers":
		a.push("virtualservers", func() tview.Primitive { return a.virtualServersView() })
	case "pool", "pools":
		a.push("pools", func() tview.Primitive { return a.poolsView() })
	case "pol", "policy", "policies":
		a.push("policies", func() tview.Primitive { return a.ltmPoliciesView() })
	case "asm":
		a.push("asm", func() tview.Primitive { return a.asmPoliciesView() })
	case "help":
		a.push("help", func() tview.Primitive { return helpView() })
	case "part", "partition", "p":
		if len(args) == 0 || args[0] == "*" || args[0] == "all" {
			a.partition = ""
		} else {
			a.partition = args[0]
		}
		a.refresh()
	case "clear":
		a.filter = ""
		a.refresh()
	default:
		a.flash(fmt.Sprintf("unknown command: %s", cmd))
		return
	}
	a.updateStatus("")
}

func (a *App) push(name string, builder func() tview.Primitive) {
	a.stack = append(a.stack, frame{name: name, builder: builder})
	a.render()
}

func (a *App) refresh() {
	if len(a.stack) == 0 {
		return
	}
	a.render()
	a.updateStatus("")
}

func (a *App) back() {
	if len(a.stack) < 2 {
		return
	}
	top := a.stack[len(a.stack)-1]
	a.stack = a.stack[:len(a.stack)-1]
	if a.pages.HasPage(top.name) {
		a.pages.RemovePage(top.name)
	}
	a.render()
}

func (a *App) render() {
	if len(a.stack) == 0 {
		return
	}
	top := a.stack[len(a.stack)-1]
	p := top.builder()
	if a.pages.HasPage(top.name) {
		a.pages.RemovePage(top.name)
	}
	a.pages.AddPage(top.name, p, true, true)
	a.tv.SetFocus(p)
}

func (a *App) updateStatus(extra string) {
	part := a.partition
	if part == "" {
		part = "*"
	}
	filt := a.filter
	if filt == "" {
		filt = "-"
	}
	line := fmt.Sprintf(" [yellow]part[-]:%s  [yellow]filter[-]:%s  [yellow]:[-]cmd [yellow]/[-]filter [yellow]?[-]help [yellow]esc[-]back [yellow]:q[-]quit ",
		part, filt)
	if extra != "" {
		line = " [red]" + extra + "[-]  " + line
	}
	a.status.SetText(line)
}

func (a *App) flash(msg string) {
	a.updateStatus(msg)
}

// matchPartition returns true if the item's partition passes the current filter.
// Empty partition means "all".
func (a *App) matchPartition(p string) bool {
	return a.partition == "" || strings.EqualFold(p, a.partition)
}

// matchFilter returns true if any of the provided fields contain the filter substring.
// Empty filter matches everything.
func (a *App) matchFilter(fields ...string) bool {
	if a.filter == "" {
		return true
	}
	needle := strings.ToLower(a.filter)
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	return false
}

func helpView() tview.Primitive {
	tv := tview.NewTextView().SetDynamicColors(true)
	tv.SetBorder(true).SetTitle(" help ")
	tv.SetText(`
 [yellow]Commands[-]
   :vs                virtual servers
   :pools             pools
   :policies          LTM policies
   :asm               ASM policies
   :part <name>       switch partition (blank / * = all)
   :clear             clear the row filter
   :q                 quit

 [yellow]Keys[-]
   :                  open command bar
   /                  filter rows in the current view
   ?                  this help
   esc                go back / clear filter input
   enter              drill into selection (pools only)
`)
	return tv
}
