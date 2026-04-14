package ui

import (
	"fmt"

	"f5tui/internal/f5"

	"github.com/rivo/tview"
)

// detailRow is one line in a details table. If onEnter is nil the row is
// unselectable (section header, static key/value pair, informational line).
type detailRow struct {
	cells   []string
	onEnter func()
}

func kv(k, v string) detailRow {
	return detailRow{cells: []string{"[yellow]" + k + "[-]", v}}
}

func section(title string) detailRow {
	return detailRow{cells: []string{"[::b]" + title + "[::-]"}}
}

func blank() detailRow { return detailRow{cells: []string{""}} }

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// buildDetailTable renders a mix of key/value rows, section headers and
// selectable linked rows into a tview.Table. Selectable rows store their
// onEnter closure on the reference of their first cell.
func buildDetailTable(title string, rows []detailRow) *tview.Table {
	t := tview.NewTable().SetBorders(false).SetSelectable(true, false)
	t.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", title))

	cols := 1
	for _, r := range rows {
		if len(r.cells) > cols {
			cols = len(r.cells)
		}
	}
	for i, row := range rows {
		cells := row.cells
		for len(cells) < cols {
			cells = append(cells, "")
		}
		for c, text := range cells {
			cell := tview.NewTableCell(text).SetExpansion(1)
			if row.onEnter == nil {
				cell.SetSelectable(false)
			}
			t.SetCell(i, c, cell)
		}
		if row.onEnter != nil {
			t.GetCell(i, 0).SetReference(row.onEnter)
		}
	}
	t.SetSelectedFunc(func(row, _ int) {
		if fn, ok := t.GetCell(row, 0).GetReference().(func()); ok && fn != nil {
			fn()
		}
	})
	return t
}

// ---- Virtual Server detail ----

func (a *App) virtualServerDetailView(fullPath string) tview.Primitive {
	vs, err := a.client.VirtualServerDetail(fullPath)
	if err != nil {
		return errorView("virtual server "+fullPath, err)
	}

	asmAll, _ := a.client.ASMPolicies()
	var asmAttached []f5.ASMPolicy
	for _, p := range asmAll {
		for _, v := range p.VirtualServers {
			if v == fullPath {
				asmAttached = append(asmAttached, p)
				break
			}
		}
	}

	status := "[green]enabled[-]"
	if !vs.Enabled {
		status = "[red]disabled[-]"
	}

	rows := []detailRow{
		kv("Name", vs.Name),
		kv("Full path", vs.FullPath),
		kv("Partition", vs.Partition),
		kv("Destination", vs.Destination),
		kv("Protocol", vs.IPProtocol),
		kv("Source", vs.Source),
		kv("SNAT", vs.SNAT),
		kv("Status", status),
		kv("Description", vs.Description),
		blank(),
	}

	if vs.Pool != "" {
		pool := vs.Pool
		rows = append(rows,
			section("POOL"),
			detailRow{
				cells:   []string{"  " + pool, "[grey]enter[-]"},
				onEnter: func() { a.pushPool(pool) },
			},
			blank(),
		)
	}

	if n := len(vs.ProfilesReference.Items); n > 0 {
		rows = append(rows, section(fmt.Sprintf("PROFILES (%d)", n)))
		for _, p := range vs.ProfilesReference.Items {
			rows = append(rows, detailRow{cells: []string{"  " + p.FullPath, p.Context}})
		}
		rows = append(rows, blank())
	}

	if n := len(vs.PoliciesReference.Items); n > 0 {
		rows = append(rows, section(fmt.Sprintf("LTM POLICIES (%d)", n)))
		for _, p := range vs.PoliciesReference.Items {
			fp := p.FullPath
			rows = append(rows, detailRow{
				cells:   []string{"  " + fp, "[grey]enter[-]"},
				onEnter: func() { a.pushLTMPolicy(fp) },
			})
		}
		rows = append(rows, blank())
	}

	if n := len(asmAttached); n > 0 {
		rows = append(rows, section(fmt.Sprintf("ASM POLICIES (%d)", n)))
		for _, p := range asmAttached {
			id := p.ID
			name := p.Name
			rows = append(rows, detailRow{
				cells:   []string{fmt.Sprintf("  %s [grey](%s)[-]", name, id), "[grey]enter[-]"},
				onEnter: func() { a.pushASMPolicy(id) },
			})
		}
		rows = append(rows, blank())
	}

	if n := len(vs.Rules); n > 0 {
		rows = append(rows, section(fmt.Sprintf("IRULES (%d)", n)))
		for _, r := range vs.Rules {
			rows = append(rows, detailRow{cells: []string{"  " + r}})
		}
	}

	return buildDetailTable("virtual server: "+vs.Name, rows)
}

// ---- Pool detail ----

func (a *App) poolDetailView(fullPath string) tview.Primitive {
	p, err := a.client.PoolDetail(fullPath)
	if err != nil {
		return errorView("pool "+fullPath, err)
	}
	rows := []detailRow{
		kv("Name", p.Name),
		kv("Full path", p.FullPath),
		kv("Partition", p.Partition),
		kv("Monitor", p.Monitor),
		kv("LB mode", p.LoadBalancingMode),
		kv("Active members", fmt.Sprintf("%d", p.ActiveMemberCount)),
		kv("Description", p.Description),
		blank(),
		section(fmt.Sprintf("MEMBERS (%d)", len(p.Members))),
	}
	for _, m := range p.Members {
		state := m.State
		switch state {
		case "up":
			state = "[green]up[-]"
		case "down":
			state = "[red]down[-]"
		}
		rows = append(rows, detailRow{cells: []string{"  " + m.Name, m.Address, state, m.Session}})
	}
	return buildDetailTable("pool: "+p.Name, rows)
}

// ---- LTM policy detail + rule detail ----

func (a *App) ltmPolicyDetailView(fullPath string) tview.Primitive {
	p, err := a.client.LTMPolicyDetail(fullPath)
	if err != nil {
		return errorView("LTM policy "+fullPath, err)
	}
	status := p.Status
	switch status {
	case "published":
		status = "[green]published[-]"
	case "draft":
		status = "[yellow]draft[-]"
	}
	rows := []detailRow{
		kv("Name", p.Name),
		kv("Full path", p.FullPath),
		kv("Partition", p.Partition),
		kv("Status", status),
		kv("Strategy", p.Strategy),
		kv("Description", p.Description),
		blank(),
		section(fmt.Sprintf("RULES (%d) — enter to see conditions + actions", len(p.RulesReference.Items))),
	}
	for i := range p.RulesReference.Items {
		rule := p.RulesReference.Items[i]
		policyPath := p.FullPath
		rows = append(rows, detailRow{
			cells: []string{
				fmt.Sprintf("  %d. %s", rule.Ordinal, rule.Name),
				fmt.Sprintf("%d cond / %d act", len(rule.ConditionsReference.Items), len(rule.ActionsReference.Items)),
				"[grey]enter[-]",
			},
			onEnter: func() { a.pushLTMRule(policyPath, rule) },
		})
	}
	return buildDetailTable("policy: "+p.Name, rows)
}

func (a *App) ltmRuleDetailView(policyPath string, rule f5.PolicyRule) tview.Primitive {
	rows := []detailRow{
		kv("Policy", policyPath),
		kv("Rule", rule.Name),
		kv("Ordinal", fmt.Sprintf("%d", rule.Ordinal)),
		kv("Description", rule.Description),
		blank(),
		section(fmt.Sprintf("CONDITIONS (%d)", len(rule.ConditionsReference.Items))),
	}
	for _, c := range rule.ConditionsReference.Items {
		rows = append(rows, detailRow{cells: []string{"  " + c.Describe()}})
	}
	rows = append(rows, blank(), section(fmt.Sprintf("ACTIONS (%d)", len(rule.ActionsReference.Items))))
	for _, act := range rule.ActionsReference.Items {
		row := detailRow{cells: []string{"  " + act.Describe()}}
		if act.Forward && act.Pool != "" {
			pool := act.Pool
			row.cells = append(row.cells, "[grey]enter → pool[-]")
			row.onEnter = func() { a.pushPool(pool) }
		}
		rows = append(rows, row)
	}
	return buildDetailTable("rule: "+rule.Name, rows)
}

// ---- ASM policy detail + subcollections ----

func (a *App) asmPolicyDetailView(id string) tview.Primitive {
	p, err := a.client.ASMPolicyDetail(id)
	if err != nil {
		return errorView("ASM policy "+id, err)
	}
	enforcement := p.EnforcementMode
	switch enforcement {
	case "blocking":
		enforcement = "[red]blocking[-]"
	case "transparent":
		enforcement = "[yellow]transparent[-]"
	}
	active := "[red]no[-]"
	if p.Active {
		active = "[green]yes[-]"
	}

	rows := []detailRow{
		kv("ID", p.ID),
		kv("Name", p.Name),
		kv("Partition", p.Partition),
		kv("Enforcement", enforcement),
		kv("Active", active),
		kv("Learning mode", p.LearningMode),
		kv("App language", p.ApplicationLanguage),
		kv("Case-insensitive", yesno(p.CaseInsensitive)),
		kv("Passive mode", yesno(p.EnablePassiveMode)),
		kv("Protocol-independent", yesno(p.ProtocolIndependent)),
		kv("Signature staging", yesno(p.SignatureStaging)),
		kv("Description", p.Description),
		blank(),
	}

	if n := len(p.VirtualServers); n > 0 {
		rows = append(rows, section(fmt.Sprintf("ATTACHED VIRTUAL SERVERS (%d)", n)))
		for _, vs := range p.VirtualServers {
			fp := vs
			rows = append(rows, detailRow{
				cells:   []string{"  " + fp, "[grey]enter[-]"},
				onEnter: func() { a.pushVirtualServer(fp) },
			})
		}
		rows = append(rows, blank())
	}

	if n := len(p.SignatureSets); n > 0 {
		rows = append(rows, section(fmt.Sprintf("SIGNATURE SETS (%d)", n)))
		for _, s := range p.SignatureSets {
			alarm := "-"
			if s.Alarm {
				alarm = "alarm"
			}
			block := "-"
			if s.Block {
				block = "block"
			}
			rows = append(rows, detailRow{cells: []string{"  " + s.Name, alarm, block}})
		}
		rows = append(rows, blank())
	}

	rows = append(rows,
		section("SUB-COLLECTIONS"),
		detailRow{
			cells:   []string{"  URLs", "[grey]enter[-]"},
			onEnter: func() { a.pushASMURLs(id, p.Name) },
		},
		detailRow{
			cells:   []string{"  Parameters", "[grey]enter[-]"},
			onEnter: func() { a.pushASMParameters(id, p.Name) },
		},
	)

	return buildDetailTable("ASM policy: "+p.Name, rows)
}

func (a *App) asmURLsView(id, policyName string) tview.Primitive {
	urls, err := a.client.ASMPolicyURLs(id)
	if err != nil {
		return errorView("ASM URLs "+id, err)
	}
	rows := []detailRow{
		kv("ASM policy", policyName),
		kv("ID", id),
		blank(),
		section(fmt.Sprintf("URLs (%d)", len(urls))),
		{cells: []string{"  URL", "PROTOCOL", "METHOD", "TYPE", "STAGING"}},
	}
	shown := 0
	for _, u := range urls {
		if !a.matchFilter(u.Name, u.Method, u.Protocol, u.Type) {
			continue
		}
		staging := "-"
		if u.PerformStaging {
			staging = "yes"
		}
		rows = append(rows, detailRow{cells: []string{"  " + u.Name, u.Protocol, u.Method, u.Type, staging}})
		shown++
	}
	if shown == 0 && len(urls) > 0 {
		rows = append(rows, detailRow{cells: []string{"  (no rows match filter)"}})
	}
	return buildDetailTable(fmt.Sprintf("ASM URLs: %s", policyName), rows)
}

func (a *App) asmParametersView(id, policyName string) tview.Primitive {
	params, err := a.client.ASMPolicyParameters(id)
	if err != nil {
		return errorView("ASM parameters "+id, err)
	}
	rows := []detailRow{
		kv("ASM policy", policyName),
		kv("ID", id),
		blank(),
		section(fmt.Sprintf("PARAMETERS (%d)", len(params))),
		{cells: []string{"  NAME", "TYPE", "LEVEL", "VALUE-TYPE", "STAGING"}},
	}
	shown := 0
	for _, p := range params {
		if !a.matchFilter(p.Name, p.Type, p.Level, p.ValueType) {
			continue
		}
		staging := "-"
		if p.PerformStaging {
			staging = "yes"
		}
		rows = append(rows, detailRow{cells: []string{"  " + p.Name, p.Type, p.Level, p.ValueType, staging}})
		shown++
	}
	if shown == 0 && len(params) > 0 {
		rows = append(rows, detailRow{cells: []string{"  (no rows match filter)"}})
	}
	return buildDetailTable(fmt.Sprintf("ASM parameters: %s", policyName), rows)
}

// ---- navigation helpers (keep one place for the stack-frame naming) ----

func (a *App) pushVirtualServer(fullPath string) {
	a.push("vs:"+fullPath, func() tview.Primitive { return a.virtualServerDetailView(fullPath) })
}
func (a *App) pushPool(fullPath string) {
	a.push("pool:"+fullPath, func() tview.Primitive { return a.poolDetailView(fullPath) })
}
func (a *App) pushLTMPolicy(fullPath string) {
	a.push("policy:"+fullPath, func() tview.Primitive { return a.ltmPolicyDetailView(fullPath) })
}
func (a *App) pushLTMRule(policyPath string, rule f5.PolicyRule) {
	a.push("rule:"+policyPath+"#"+rule.Name, func() tview.Primitive { return a.ltmRuleDetailView(policyPath, rule) })
}
func (a *App) pushASMPolicy(id string) {
	a.push("asm:"+id, func() tview.Primitive { return a.asmPolicyDetailView(id) })
}
func (a *App) pushASMURLs(id, policyName string) {
	a.push("asm_urls:"+id, func() tview.Primitive { return a.asmURLsView(id, policyName) })
}
func (a *App) pushASMParameters(id, policyName string) {
	a.push("asm_params:"+id, func() tview.Primitive { return a.asmParametersView(id, policyName) })
}
