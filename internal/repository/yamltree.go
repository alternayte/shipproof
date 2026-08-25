package repository

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// yamlTree is a minimal ordered document for the small YAML subset that
// .shipproof/config.yaml uses: nested mappings and scalar string values.
// It preserves key order, comments, and blank lines. It stores a sequence
// under a key as an opaque, ordered list of raw lines, and it reproduces
// that list unchanged on render. It does not read or write a sequence
// item. It does not support anchors or multi-line scalars. v0 needs none
// of those.
type yamlTree struct {
	nodes    []*yamlNode
	trailing []string
}

type yamlNode struct {
	// lead holds comment and blank lines that precede this key.
	lead     []string
	key      string
	value    string
	mapping  bool
	children []*yamlNode
	// sequenceItems holds a raw "- item" list under this key. get and set do
	// not read or write a sequence. render reproduces it unchanged.
	sequenceItems []string
}

const emptyMapping = "{}"

func parseTree(reader io.Reader) (*yamlTree, error) {
	tree := &yamlTree{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// stack holds the open parent at each indentation level.
	type frame struct {
		indent int
		node   *yamlNode
	}
	var stack []frame
	var lead []string

	appendChild := func(indent int, node *yamlNode) error {
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			if indent != 0 {
				return fmt.Errorf("unexpected indentation on %q", node.key)
			}
			tree.nodes = append(tree.nodes, node)
		} else {
			parent := stack[len(stack)-1].node
			parent.mapping = true
			parent.value = ""
			parent.children = append(parent.children, node)
		}
		stack = append(stack, frame{indent: indent, node: node})
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lead = append(lead, trimmed)
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if item, isItem := strings.CutPrefix(trimmed, "- "); isItem {
			if len(stack) == 0 {
				return nil, fmt.Errorf("unexpected sequence item %q", trimmed)
			}
			parent := stack[len(stack)-1].node
			parent.sequenceItems = append(parent.sequenceItems, item)
			lead = nil
			continue
		}
		key, value, ok := splitKV(trimmed)
		if !ok {
			return nil, fmt.Errorf("cannot parse config line %q", trimmed)
		}
		node := &yamlNode{lead: lead, key: key, value: value, mapping: value == "" || value == emptyMapping}
		lead = nil
		if err := appendChild(indent, node); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	tree.trailing = lead
	return tree, nil
}

func (tree *yamlTree) get(path []string) (string, bool) {
	nodes := tree.nodes
	for depth, key := range path {
		node := findNode(nodes, key)
		if node == nil {
			return "", false
		}
		if depth == len(path)-1 {
			if node.mapping && node.value != emptyMapping {
				return "", false
			}
			return node.value, true
		}
		nodes = node.children
	}
	return "", false
}

func (tree *yamlTree) set(path []string, value string) {
	nodes := &tree.nodes
	var node *yamlNode
	for depth, key := range path {
		node = findNode(*nodes, key)
		if node == nil {
			node = &yamlNode{key: key}
			*nodes = append(*nodes, node)
		}
		if depth == len(path)-1 {
			node.value = value
			node.mapping = false
			node.children = nil
			return
		}
		node.mapping = true
		if node.value == emptyMapping {
			node.value = ""
		}
		nodes = &node.children
	}
}

// keys returns the child key names under a path in document order.
func (tree *yamlTree) keys(path []string) []string {
	nodes := tree.nodes
	for _, key := range path {
		node := findNode(nodes, key)
		if node == nil {
			return nil
		}
		nodes = node.children
	}
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		names = append(names, node.key)
	}
	return names
}

func findNode(nodes []*yamlNode, key string) *yamlNode {
	for _, node := range nodes {
		if node.key == key {
			return node
		}
	}
	return nil
}

func (tree *yamlTree) render() string {
	var builder strings.Builder
	renderNodes(&builder, tree.nodes, 0)
	for _, line := range tree.trailing {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

func renderNodes(builder *strings.Builder, nodes []*yamlNode, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, node := range nodes {
		for _, line := range node.lead {
			if line == "" {
				builder.WriteString("\n")
				continue
			}
			builder.WriteString(indent)
			builder.WriteString(line)
			builder.WriteString("\n")
		}
		builder.WriteString(indent)
		builder.WriteString(node.key)
		builder.WriteString(":")
		switch {
		case len(node.children) > 0:
			builder.WriteString("\n")
			renderNodes(builder, node.children, depth+1)
		case len(node.sequenceItems) > 0:
			builder.WriteString("\n")
			itemIndent := strings.Repeat("  ", depth+1)
			for _, item := range node.sequenceItems {
				builder.WriteString(itemIndent)
				builder.WriteString("- ")
				builder.WriteString(item)
				builder.WriteString("\n")
			}
		case node.mapping && node.value == emptyMapping:
			builder.WriteString(" {}\n")
		case node.value == "":
			builder.WriteString(" \"\"\n")
		default:
			builder.WriteString(" ")
			builder.WriteString(quoteIfNeeded(node.value))
			builder.WriteString("\n")
		}
	}
}

func quoteIfNeeded(value string) string {
	if strings.ContainsAny(value, "#:\"'") || strings.TrimSpace(value) != value {
		return "\"" + strings.ReplaceAll(value, `"`, `\"`) + "\""
	}
	return value
}
