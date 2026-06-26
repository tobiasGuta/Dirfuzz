import re

def process():
    path = r"d:\Tools\DirFuzz-Mcp-Monitor\pkg\tui\tui.go"
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()

    # Add StateGraph
    content = content.replace(
        "StateRepeater\n)",
        "StateRepeater\n\tStateGraph\n)"
    )

    # Add toggleGraphView function
    if "toggleGraphView" not in content:
        toggle_fn = """
func (m *Model) toggleGraphView() {
	if m.state == StateGraph {
		m.state = m.previousState
		m.cmdViewport.GotoBottom()
	} else {
		m.previousState = m.state
		m.state = StateGraph
		m.cmdViewport.GotoBottom()
	}
}
"""
        content = content.replace("func (m *Model) toggleLogsPanel() {", toggle_fn + "func (m *Model) toggleLogsPanel() {")

    # Add keyboard shortcut 'g'
    content = content.replace(
        """		case "L":
			m.toggleLogsPanel()
			return m, nil""",
        """		case "g", "G":
			m.toggleGraphView()
			return m, nil
		case "L":
			m.toggleLogsPanel()
			return m, nil"""
    )

    # Add rendering for StateGraph
    render_graph = """	} else if activeState == StateGraph {
		vpHeight := remainingHeight
		vpHeight -= actualLogPanelHeight
		if vpHeight < 5 {
			vpHeight = 5
		}
		graphPaneWidth := m.width - 2
		if graphPaneWidth < 20 {
			graphPaneWidth = 20
		}
		
		header := renderPaneHeader(requestPaneHeaderStyle, graphPaneWidth, "🕸  Discovery Graph")
		separator := separatorStyle.Render(strings.Repeat("─", graphPaneWidth))
		
		graphContent := m.renderGraphView()
		paddedContent := lipgloss.NewStyle().PaddingLeft(2).Render(graphContent)
		
		// Create a temporary viewport just to clip/scroll the graph
		vp := viewport.New(graphPaneWidth, vpHeight-2)
		vp.SetContent(paddedContent)
		vp.YOffset = m.listScrollIdx // Use list scroll index for now
		
		graphPane := paneStyle.Width(graphPaneWidth).Height(vpHeight - 2).Render(
			lipgloss.JoinVertical(lipgloss.Top,
				header,
				separator,
				vp.View(),
			),
		)
		mainContent = graphPane
	}

	// Footer"""
    content = content.replace("	}\n\n	// Footer", render_graph)

    # Add leftChips for StateGraph
    content = content.replace(
        """		} else if m.state == StateCommand {
			leftChips = m.textInput.View()
		}""",
        """		} else if m.state == StateGraph {
			leftChips = strings.Join([]string{
				keyChip("L", "logs"),
				keyChip("Esc/q/g", "back"),
				keyChip("Up/Down", "scroll"),
			}, "  ")
		} else if m.state == StateCommand {
			leftChips = m.textInput.View()
		}"""
    )
    
    # Also add "g/G" to StateList leftChips
    content = content.replace(
        """				keyChip("d", "diff"),
				keyChip("R", "bookmark"),""",
        """				keyChip("d", "diff"),
				keyChip("R", "bookmark"),
				keyChip("g", "graph"),"""
    )

    # Add the actual renderGraphView() func
    render_func = """
func (m *Model) renderGraphView() string {
	if m.Engine == nil || m.Engine.DiscoveryGraph == nil {
		return "Graph not available."
	}
	
	snap := m.Engine.DiscoveryGraph.GetSnapshot()
	if len(snap.Nodes) == 0 {
		return "Graph is empty."
	}

	// Find root nodes
	var roots []*engine.DiscoveryNode
	for _, n := range snap.Nodes {
		if n.Kind == engine.NodeSource || n.ParentID == "" {
			roots = append(roots, n)
		}
	}

	// Sort roots by first seen or label
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Label < roots[j].Label
	})
	
	// Create map of current scan results by node ID
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
		
		// Determine color based on results
		labelColor := lipgloss.NewStyle().Foreground(DraculaPurple) // Default for path
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
		
		// Add evidence or extra info
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
"""
    if "renderGraphView()" not in content:
        content += render_func

    with open(path, "w", encoding="utf-8") as f:
        f.write(content)

process()
