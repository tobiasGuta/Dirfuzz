import re

with open('pkg/tui/tui.go', 'a', encoding='utf-8') as f:
    f.write('''
func (m *Model) renderGraphView() string {
	if m.Engine == nil || m.Engine.DiscoveryGraph == nil {
		return "Graph not available."
	}
	
	snap := m.Engine.DiscoveryGraph.GetSnapshot()
	if len(snap.Nodes) == 0 {
		return "Graph is empty."
	}

	var roots []*engine.DiscoveryNode
	for _, n := range snap.Nodes {
		if n.Kind == engine.NodeSource || n.ParentID == "" {
			roots = append(roots, n)
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Label < roots[j].Label
	})
	
	resultsByNode := make(map[string]engine.Result)
	for _, r := range m.hits {
		if r.DiscoveryNodeID != "" {
			resultsByNode[r.DiscoveryNodeID] = r
		}
	}
	
	var sb strings.Builder
	var renderNode func(n *engine.DiscoveryNode, prefix string, isLast bool)
	renderNode = func(n *engine.DiscoveryNode, prefix string, isLast bool) {
		marker := "├─ "
		childPrefix := prefix + "│  "
		if isLast {
			marker = "└─ "
			childPrefix = prefix + "   "
		}
		
		if n.ParentID == "" {
			marker = ""
			childPrefix = ""
		}
		
		line := prefix + marker
		
		labelColor := lipgloss.NewStyle().Foreground(DraculaPurple)
		if n.Kind == engine.NodeSource {
			labelColor = lipgloss.NewStyle().Foreground(DraculaCyan).Bold(true)
		}
		
		if res, ok := resultsByNode[n.ID]; ok {
			switch {
			case res.StatusCode >= 200 && res.StatusCode < 300:
				labelColor = status2xxStyle
			case res.StatusCode >= 300 && res.StatusCode < 400:
				labelColor = status3xxStyle
			case res.StatusCode == 403:
				labelColor = status403Style
			case res.StatusCode >= 400 && res.StatusCode < 500:
				labelColor = status4xxStyle
			case res.StatusCode >= 500:
				labelColor = status5xxStyle
			}
		}
		
		line += labelColor.Render(n.Label)
		
		if n.Kind == engine.NodePath {
			var extras []string
			if res, ok := resultsByNode[n.ID]; ok {
				extras = append(extras, fmt.Sprintf("Status: %d", res.StatusCode))
			}
			if n.Evidence.Type != "" {
				extras = append(extras, mutedStyle.Render(fmt.Sprintf("[%s]", n.Evidence.Type)))
			}
			if len(extras) > 0 {
				line += " " + strings.Join(extras, " ")
			}
		}
		
		sb.WriteString(line + "\\n")
		
		var children []*engine.DiscoveryNode
		for _, childID := range n.Children {
			if child, ok := snap.Nodes[childID]; ok {
				children = append(children, child)
			}
		}
		sort.Slice(children, func(i, j int) bool {
			return children[i].Label < children[j].Label
		})
		
		for i, child := range children {
			renderNode(child, childPrefix, i == len(children)-1)
		}
	}
	
	for i, root := range roots {
		renderNode(root, "", false)
		if i < len(roots)-1 {
			sb.WriteString("\\n")
		}
	}
	
	return sb.String()
}
''')
