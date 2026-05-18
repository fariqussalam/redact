package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AddListValue adds value to a YAML sequence and creates the config file if needed.
func AddListValue(path, key, value string) (bool, error) {
	doc, err := loadNode(path)
	if err != nil {
		return false, err
	}
	seq := ensureSeq(doc, key)
	for _, n := range seq.Content {
		if n.Value == value {
			return false, saveNode(path, doc)
		}
	}
	seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
	return true, saveNode(path, doc)
}

// RemoveListValue removes value from a YAML sequence if present.
func RemoveListValue(path, key, value string) (bool, error) {
	doc, err := loadNode(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	seq := findSeq(doc, key)
	if seq == nil {
		return false, nil
	}
	out := seq.Content[:0]
	removed := false
	for _, n := range seq.Content {
		if n.Value == value {
			removed = true
			continue
		}
		out = append(out, n)
	}
	seq.Content = out
	if !removed {
		return false, nil
	}
	return true, saveNode(path, doc)
}

func loadNode(path string) (*yaml.Node, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		use := true
		mask := DefaultMask
		cfg := Config{UseDefaults: &use, Mask: &mask, Names: []string{}, Patterns: []string{}}
		var n yaml.Node
		if err := n.Encode(cfg); err != nil {
			return nil, err
		}
		return &n, nil
	}
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(b, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func saveNode(path string, n *yaml.Node) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func mapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

func findSeq(doc *yaml.Node, key string) *yaml.Node {
	m := mapping(doc)
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key && m.Content[i+1].Kind == yaml.SequenceNode {
			return m.Content[i+1]
		}
	}
	return nil
}

func ensureSeq(doc *yaml.Node, key string) *yaml.Node {
	if s := findSeq(doc, key); s != nil {
		return s
	}
	m := mapping(doc)
	if m.Kind != yaml.MappingNode {
		m.Kind = yaml.MappingNode
		m.Tag = "!!map"
	}
	s := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, s)
	return s
}
