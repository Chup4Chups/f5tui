package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newTable(title string, headers []string) *tview.Table {
	t := tview.NewTable().SetBorders(false).SetSelectable(true, false).SetFixed(1, 0)
	t.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", title))
	for i, h := range headers {
		t.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1))
	}
	return t
}

func errorView(title string, err error) tview.Primitive {
	tv := tview.NewTextView().SetText(fmt.Sprintf("error loading %s:\n\n%v", title, err))
	tv.SetBorder(true).SetTitle(fmt.Sprintf(" %s (error) ", title))
	return tv
}

func (a *App) virtualServersView() tview.Primitive {
	items, err := a.client.VirtualServers()
	if err != nil {
		return errorView("virtual servers", err)
	}
	t := newTable("", []string{"NAME", "PARTITION", "DESTINATION", "PROTOCOL", "POOL", "STATUS"})
	row := 1
	shown := 0
	for _, v := range items {
		if !a.matchPartition(v.Partition) {
			continue
		}
		if !a.matchFilter(v.Name, v.FullPath, v.Destination, v.Pool) {
			continue
		}
		status := "[green]enabled[-]"
		if !v.Enabled {
			status = "[red]disabled[-]"
		}
		t.SetCell(row, 0, tview.NewTableCell(v.Name).SetExpansion(1).SetReference(v.FullPath))
		t.SetCell(row, 1, tview.NewTableCell(v.Partition).SetExpansion(1))
		t.SetCell(row, 2, tview.NewTableCell(v.Destination).SetExpansion(2))
		t.SetCell(row, 3, tview.NewTableCell(v.IPProtocol).SetExpansion(1))
		t.SetCell(row, 4, tview.NewTableCell(v.Pool).SetExpansion(2))
		t.SetCell(row, 5, tview.NewTableCell(status).SetExpansion(1))
		row++
		shown++
	}
	t.SetTitle(fmt.Sprintf(" virtual servers (%d/%d) — enter for details ", shown, len(items)))
	t.SetSelectedFunc(func(r, _ int) {
		if fp, ok := t.GetCell(r, 0).GetReference().(string); ok && fp != "" {
			a.pushVirtualServer(fp)
		}
	})
	return t
}

func (a *App) poolsView() tview.Primitive {
	items, err := a.client.Pools()
	if err != nil {
		return errorView("pools", err)
	}
	t := newTable("", []string{"NAME", "PARTITION", "LB MODE", "MONITOR", "ACTIVE"})
	row := 1
	shown := 0
	for _, p := range items {
		if !a.matchPartition(p.Partition) {
			continue
		}
		if !a.matchFilter(p.Name, p.FullPath, p.Monitor) {
			continue
		}
		t.SetCell(row, 0, tview.NewTableCell(p.Name).SetExpansion(1).SetReference(p.FullPath))
		t.SetCell(row, 1, tview.NewTableCell(p.Partition).SetExpansion(1))
		t.SetCell(row, 2, tview.NewTableCell(p.LoadBalancingMode).SetExpansion(1))
		t.SetCell(row, 3, tview.NewTableCell(p.Monitor).SetExpansion(1))
		t.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", p.ActiveMemberCount)).SetExpansion(1))
		row++
		shown++
	}
	t.SetTitle(fmt.Sprintf(" pools (%d/%d) — enter for details ", shown, len(items)))
	t.SetSelectedFunc(func(r, _ int) {
		if fp, ok := t.GetCell(r, 0).GetReference().(string); ok && fp != "" {
			a.pushPool(fp)
		}
	})
	return t
}

func (a *App) ltmPoliciesView() tview.Primitive {
	items, err := a.client.LTMPolicies()
	if err != nil {
		return errorView("LTM policies", err)
	}
	t := newTable("", []string{"NAME", "PARTITION", "STATUS", "STRATEGY"})
	row := 1
	shown := 0
	for _, p := range items {
		if !a.matchPartition(p.Partition) {
			continue
		}
		if !a.matchFilter(p.Name, p.FullPath, p.Strategy) {
			continue
		}
		status := p.Status
		switch status {
		case "published":
			status = "[green]published[-]"
		case "draft":
			status = "[yellow]draft[-]"
		}
		t.SetCell(row, 0, tview.NewTableCell(p.Name).SetExpansion(1).SetReference(p.FullPath))
		t.SetCell(row, 1, tview.NewTableCell(p.Partition).SetExpansion(1))
		t.SetCell(row, 2, tview.NewTableCell(status).SetExpansion(1))
		t.SetCell(row, 3, tview.NewTableCell(p.Strategy).SetExpansion(1))
		row++
		shown++
	}
	t.SetTitle(fmt.Sprintf(" LTM policies (%d/%d) — enter for rules ", shown, len(items)))
	t.SetSelectedFunc(func(r, _ int) {
		if fp, ok := t.GetCell(r, 0).GetReference().(string); ok && fp != "" {
			a.pushLTMPolicy(fp)
		}
	})
	return t
}

func (a *App) asmPoliciesView() tview.Primitive {
	items, err := a.client.ASMPolicies()
	if err != nil {
		return errorView("ASM policies", err)
	}
	t := newTable("", []string{"NAME", "PARTITION", "ENFORCEMENT", "ACTIVE", "ATTACHED VS"})
	row := 1
	shown := 0
	for _, p := range items {
		if !a.matchPartition(p.Partition) {
			continue
		}
		vs := strings.Join(p.VirtualServers, ", ")
		if !a.matchFilter(p.Name, p.Partition, vs) {
			continue
		}
		enforce := p.EnforcementMode
		switch enforce {
		case "blocking":
			enforce = "[red]blocking[-]"
		case "transparent":
			enforce = "[yellow]transparent[-]"
		}
		active := "[red]no[-]"
		if p.Active {
			active = "[green]yes[-]"
		}
		if vs == "" {
			vs = "-"
		}
		t.SetCell(row, 0, tview.NewTableCell(p.Name).SetExpansion(1).SetReference(p.ID))
		t.SetCell(row, 1, tview.NewTableCell(p.Partition).SetExpansion(1))
		t.SetCell(row, 2, tview.NewTableCell(enforce).SetExpansion(1))
		t.SetCell(row, 3, tview.NewTableCell(active).SetExpansion(1))
		t.SetCell(row, 4, tview.NewTableCell(vs).SetExpansion(2))
		row++
		shown++
	}
	t.SetTitle(fmt.Sprintf(" ASM policies (%d/%d) — enter for details ", shown, len(items)))
	t.SetSelectedFunc(func(r, _ int) {
		if id, ok := t.GetCell(r, 0).GetReference().(string); ok && id != "" {
			a.pushASMPolicy(id)
		}
	})
	return t
}
