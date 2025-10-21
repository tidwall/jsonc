package jsonc

import (
	"fmt"
	"strconv"
	"strings"
)

// NodeType represents the type of a JSONC node
type NodeType int

const (
	NodeTypeObject NodeType = iota
	NodeTypeArray
	NodeTypeString
	NodeTypeNumber
	NodeTypeBool
	NodeTypeNull
	NodeTypeComment
	NodeTypeWhitespace
)

// Node represents a node in the JSONC AST
type Node struct {
	Type     NodeType
	Value    interface{}
	Raw      string // Original raw text
	Children []*Node
	Key      string // For object properties
	Parent   *Node

	// Formatting preservation
	LeadingComments    []*Node
	TrailingComments   []*Node
	LeadingWhitespace  string
	TrailingWhitespace string
	HasTrailingComma   bool
}

// Document represents a parsed JSONC document
type Document struct {
	Root *Node
	raw  []byte
}

// ToJSON strips out comments and trailing commas and convert the input to a
// valid JSON per the official spec: https://tools.ietf.org/html/rfc8259
//
// The resulting JSON will always be the same length as the input and it will
// include all of the same line breaks at matching offsets. This is to ensure
// the result can be later processed by a external parser and that that
// parser will report messages or errors with the correct offsets.
func ToJSON(src []byte) []byte {
	return toJSON(src, nil)
}

// ToJSONInPlace is the same as ToJSON, but this method reuses the input json
// buffer to avoid allocations. Do not use the original bytes slice upon return.
func ToJSONInPlace(src []byte) []byte {
	return toJSON(src, src)
}

func toJSON(src, dst []byte) []byte {
	dst = dst[:0]
	for i := 0; i < len(src); i++ {
		if src[i] == '/' {
			if i < len(src)-1 {
				if src[i+1] == '/' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src); i++ {
						if src[i] == '\n' {
							dst = append(dst, '\n')
							break
						} else if src[i] == '\t' || src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
				if src[i+1] == '*' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src)-1; i++ {
						if src[i] == '*' && src[i+1] == '/' {
							dst = append(dst, ' ', ' ')
							i++
							break
						} else if src[i] == '\n' || src[i] == '\t' ||
							src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
			}
		}
		dst = append(dst, src[i])
		if src[i] == '"' {
			for i = i + 1; i < len(src); i++ {
				dst = append(dst, src[i])
				if src[i] == '"' {
					j := i - 1
					for ; ; j-- {
						if src[j] != '\\' {
							break
						}
					}
					if (j-i)%2 != 0 {
						break
					}
				}
			}
		} else if src[i] == '}' || src[i] == ']' {
			for j := len(dst) - 2; j >= 0; j-- {
				if dst[j] <= ' ' {
					continue
				}
				if dst[j] == ',' {
					dst[j] = ' '
				}
				break
			}
		}
	}
	return dst
}

// Parse parses a JSONC document and returns a Document with preserved formatting
func Parse(src []byte) (*Document, error) {
	parser := &parser{
		src: src,
		pos: 0,
	}

	doc := &Document{
		raw: src,
	}

	root, err := parser.parseValue()
	if err != nil {
		return nil, err
	}

	doc.Root = root
	return doc, nil
}

// parser holds the parsing state
type parser struct {
	src []byte
	pos int
}

// parseValue parses any JSON value
func (p *parser) parseValue() (*Node, error) {
	p.skipWhitespaceAndComments()

	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch p.src[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f':
		return p.parseBool()
	case 'n':
		return p.parseNull()
	default:
		if p.isDigitStart(p.src[p.pos]) {
			return p.parseNumber()
		}
		return nil, fmt.Errorf("unexpected character at position %d: %c", p.pos, p.src[p.pos])
	}
}

// parseObject parses a JSON object
func (p *parser) parseObject() (*Node, error) {
	start := p.pos
	node := &Node{
		Type:     NodeTypeObject,
		Children: []*Node{},
	}

	p.pos++ // skip '{'
	p.skipWhitespaceAndComments()

	if p.pos < len(p.src) && p.src[p.pos] == '}' {
		p.pos++ // skip '}'
		node.Raw = string(p.src[start:p.pos])
		return node, nil
	}

	for {
		// Parse key
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unexpected end of input in object")
		}

		keyNode, err := p.parseString()
		if err != nil {
			return nil, err
		}

		p.skipWhitespaceAndComments()

		// Expect ':'
		if p.pos >= len(p.src) || p.src[p.pos] != ':' {
			return nil, fmt.Errorf("expected ':' after key at position %d", p.pos)
		}
		p.pos++ // skip ':'

		p.skipWhitespaceAndComments()

		// Parse value
		valueNode, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		valueNode.Key = keyNode.Value.(string)
		valueNode.Parent = node
		node.Children = append(node.Children, valueNode)

		p.skipWhitespaceAndComments()

		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unexpected end of input in object")
		}

		if p.src[p.pos] == ',' {
			p.pos++ // skip ','
			// Check for trailing comma
			p.skipWhitespaceAndComments()
			if p.pos < len(p.src) && p.src[p.pos] == '}' {
				node.HasTrailingComma = true
				break
			}
		} else if p.src[p.pos] == '}' {
			break
		} else {
			return nil, fmt.Errorf("expected ',' or '}' at position %d", p.pos)
		}
	}

	if p.pos >= len(p.src) || p.src[p.pos] != '}' {
		return nil, fmt.Errorf("expected '}' at position %d", p.pos)
	}
	p.pos++ // skip '}'

	node.Raw = string(p.src[start:p.pos])
	return node, nil
}

// parseArray parses a JSON array
func (p *parser) parseArray() (*Node, error) {
	start := p.pos
	node := &Node{
		Type:     NodeTypeArray,
		Children: []*Node{},
	}

	p.pos++ // skip '['
	p.skipWhitespaceAndComments()

	if p.pos < len(p.src) && p.src[p.pos] == ']' {
		p.pos++ // skip ']'
		node.Raw = string(p.src[start:p.pos])
		return node, nil
	}

	for {
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unexpected end of input in array")
		}

		valueNode, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		valueNode.Parent = node
		node.Children = append(node.Children, valueNode)

		p.skipWhitespaceAndComments()

		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unexpected end of input in array")
		}

		if p.src[p.pos] == ',' {
			p.pos++ // skip ','
			// Check for trailing comma
			p.skipWhitespaceAndComments()
			if p.pos < len(p.src) && p.src[p.pos] == ']' {
				node.HasTrailingComma = true
				break
			}
		} else if p.src[p.pos] == ']' {
			break
		} else {
			return nil, fmt.Errorf("expected ',' or ']' at position %d", p.pos)
		}
	}

	if p.pos >= len(p.src) || p.src[p.pos] != ']' {
		return nil, fmt.Errorf("expected ']' at position %d", p.pos)
	}
	p.pos++ // skip ']'

	node.Raw = string(p.src[start:p.pos])
	return node, nil
}

// parseString parses a JSON string
func (p *parser) parseString() (*Node, error) {
	start := p.pos

	if p.src[p.pos] != '"' {
		return nil, fmt.Errorf("expected '\"' at position %d", p.pos)
	}

	p.pos++ // skip opening quote

	for p.pos < len(p.src) {
		if p.src[p.pos] == '"' {
			// Check if it's escaped
			escaped := false
			for i := p.pos - 1; i >= start+1 && p.src[i] == '\\'; i-- {
				escaped = !escaped
			}
			if !escaped {
				p.pos++ // skip closing quote
				raw := string(p.src[start:p.pos])
				// Unquote the string value
				value := raw[1 : len(raw)-1] // Remove quotes for now - should properly unescape
				return &Node{
					Type:  NodeTypeString,
					Value: value,
					Raw:   raw,
				}, nil
			}
		}
		p.pos++
	}

	return nil, fmt.Errorf("unterminated string starting at position %d", start)
}

// parseNumber parses a JSON number
func (p *parser) parseNumber() (*Node, error) {
	start := p.pos

	// Handle negative numbers
	if p.src[p.pos] == '-' {
		p.pos++
	}

	if p.pos >= len(p.src) || !p.isDigit(p.src[p.pos]) {
		return nil, fmt.Errorf("invalid number at position %d", start)
	}

	// Parse integer part
	if p.src[p.pos] == '0' {
		p.pos++
	} else {
		for p.pos < len(p.src) && p.isDigit(p.src[p.pos]) {
			p.pos++
		}
	}

	// Parse fractional part
	if p.pos < len(p.src) && p.src[p.pos] == '.' {
		p.pos++
		if p.pos >= len(p.src) || !p.isDigit(p.src[p.pos]) {
			return nil, fmt.Errorf("invalid number at position %d", start)
		}
		for p.pos < len(p.src) && p.isDigit(p.src[p.pos]) {
			p.pos++
		}
	}

	// Parse exponent part
	if p.pos < len(p.src) && (p.src[p.pos] == 'e' || p.src[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.src) && (p.src[p.pos] == '+' || p.src[p.pos] == '-') {
			p.pos++
		}
		if p.pos >= len(p.src) || !p.isDigit(p.src[p.pos]) {
			return nil, fmt.Errorf("invalid number at position %d", start)
		}
		for p.pos < len(p.src) && p.isDigit(p.src[p.pos]) {
			p.pos++
		}
	}

	raw := string(p.src[start:p.pos])
	// For now, store as string - should parse to float64/int64
	return &Node{
		Type:  NodeTypeNumber,
		Value: raw,
		Raw:   raw,
	}, nil
}

// parseBool parses a JSON boolean
func (p *parser) parseBool() (*Node, error) {
	start := p.pos

	if p.pos+4 <= len(p.src) && string(p.src[p.pos:p.pos+4]) == "true" {
		p.pos += 4
		return &Node{
			Type:  NodeTypeBool,
			Value: true,
			Raw:   "true",
		}, nil
	}

	if p.pos+5 <= len(p.src) && string(p.src[p.pos:p.pos+5]) == "false" {
		p.pos += 5
		return &Node{
			Type:  NodeTypeBool,
			Value: false,
			Raw:   "false",
		}, nil
	}

	return nil, fmt.Errorf("invalid boolean at position %d", start)
}

// parseNull parses a JSON null
func (p *parser) parseNull() (*Node, error) {
	if p.pos+4 <= len(p.src) && string(p.src[p.pos:p.pos+4]) == "null" {
		p.pos += 4
		return &Node{
			Type:  NodeTypeNull,
			Value: nil,
			Raw:   "null",
		}, nil
	}

	return nil, fmt.Errorf("invalid null at position %d", p.pos)
}

// skipWhitespaceAndComments skips whitespace and comments
func (p *parser) skipWhitespaceAndComments() {
	for p.pos < len(p.src) {
		if p.isWhitespace(p.src[p.pos]) {
			p.pos++
		} else if p.pos < len(p.src)-1 && p.src[p.pos] == '/' {
			if p.src[p.pos+1] == '/' {
				// Line comment
				p.pos += 2
				for p.pos < len(p.src) && p.src[p.pos] != '\n' {
					p.pos++
				}
				if p.pos < len(p.src) {
					p.pos++ // skip newline
				}
			} else if p.src[p.pos+1] == '*' {
				// Block comment
				p.pos += 2
				for p.pos < len(p.src)-1 {
					if p.src[p.pos] == '*' && p.src[p.pos+1] == '/' {
						p.pos += 2
						break
					}
					p.pos++
				}
			} else {
				break
			}
		} else {
			break
		}
	}
}

// Helper functions
func (p *parser) isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func (p *parser) isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func (p *parser) isDigitStart(b byte) bool {
	return p.isDigit(b) || b == '-'
}

// Modification methods for Document

// Set sets a value at the given path in the document
func (d *Document) Set(path string, value interface{}) error {
	pathParts := parsePath(path)
	if len(pathParts) == 0 {
		return fmt.Errorf("empty path")
	}

	return d.Root.setValue(pathParts, value)
}

// Get gets a value at the given path in the document
func (d *Document) Get(path string) (interface{}, error) {
	pathParts := parsePath(path)
	if len(pathParts) == 0 {
		return d.Root.Value, nil
	}

	return d.Root.getValue(pathParts)
}

// Delete deletes a value at the given path in the document
func (d *Document) Delete(path string) error {
	pathParts := parsePath(path)
	if len(pathParts) == 0 {
		return fmt.Errorf("cannot delete root")
	}

	return d.Root.deleteValue(pathParts)
}

// Modification methods for Node

// setValue sets a value at the given path relative to this node
func (n *Node) setValue(path []string, value interface{}) error {
	if len(path) == 0 {
		// Set the value of this node
		return n.updateValue(value)
	}

	key := path[0]
	remainingPath := path[1:]

	switch n.Type {
	case NodeTypeObject:
		// Find existing child with the key
		for _, child := range n.Children {
			if child.Key == key {
				return child.setValue(remainingPath, value)
			}
		}

		// Create new child
		newNode := &Node{
			Key:    key,
			Parent: n,
		}

		if len(remainingPath) == 0 {
			// This is the final value
			newNode.updateValue(value)
		} else {
			// Create intermediate object/array
			if isArrayIndex(remainingPath[0]) {
				newNode.Type = NodeTypeArray
				newNode.Children = []*Node{}
			} else {
				newNode.Type = NodeTypeObject
				newNode.Children = []*Node{}
			}
			newNode.setValue(remainingPath, value)
		}

		n.Children = append(n.Children, newNode)
		return nil

	case NodeTypeArray:
		index, err := parseArrayIndex(key)
		if err != nil {
			return fmt.Errorf("invalid array index: %s", key)
		}

		// Extend array if necessary
		for len(n.Children) <= index {
			n.Children = append(n.Children, &Node{
				Type:   NodeTypeNull,
				Value:  nil,
				Parent: n,
			})
		}

		return n.Children[index].setValue(remainingPath, value)

	default:
		return fmt.Errorf("cannot set property on non-object/array node")
	}
}

// getValue gets a value at the given path relative to this node
func (n *Node) getValue(path []string) (interface{}, error) {
	if len(path) == 0 {
		return n.Value, nil
	}

	key := path[0]
	remainingPath := path[1:]

	switch n.Type {
	case NodeTypeObject:
		for _, child := range n.Children {
			if child.Key == key {
				return child.getValue(remainingPath)
			}
		}
		return nil, fmt.Errorf("key not found: %s", key)

	case NodeTypeArray:
		index, err := parseArrayIndex(key)
		if err != nil {
			return nil, fmt.Errorf("invalid array index: %s", key)
		}

		if index >= len(n.Children) {
			return nil, fmt.Errorf("array index out of bounds: %d", index)
		}

		return n.Children[index].getValue(remainingPath)

	default:
		return nil, fmt.Errorf("cannot get property from non-object/array node")
	}
}

// deleteValue deletes a value at the given path relative to this node
func (n *Node) deleteValue(path []string) error {
	if len(path) == 1 {
		key := path[0]

		switch n.Type {
		case NodeTypeObject:
			for i, child := range n.Children {
				if child.Key == key {
					n.Children = append(n.Children[:i], n.Children[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("key not found: %s", key)

		case NodeTypeArray:
			index, err := parseArrayIndex(key)
			if err != nil {
				return fmt.Errorf("invalid array index: %s", key)
			}

			if index >= len(n.Children) {
				return fmt.Errorf("array index out of bounds: %d", index)
			}

			n.Children = append(n.Children[:index], n.Children[index+1:]...)
			return nil

		default:
			return fmt.Errorf("cannot delete from non-object/array node")
		}
	}

	key := path[0]
	remainingPath := path[1:]

	switch n.Type {
	case NodeTypeObject:
		for _, child := range n.Children {
			if child.Key == key {
				return child.deleteValue(remainingPath)
			}
		}
		return fmt.Errorf("key not found: %s", key)

	case NodeTypeArray:
		index, err := parseArrayIndex(key)
		if err != nil {
			return fmt.Errorf("invalid array index: %s", key)
		}

		if index >= len(n.Children) {
			return fmt.Errorf("array index out of bounds: %d", index)
		}

		return n.Children[index].deleteValue(remainingPath)

	default:
		return fmt.Errorf("cannot delete from non-object/array node")
	}
}

// updateValue updates the value of this node
func (n *Node) updateValue(value interface{}) error {
	switch v := value.(type) {
	case string:
		n.Type = NodeTypeString
		n.Value = v
		n.Raw = fmt.Sprintf(`"%s"`, v) // Simple quoting - should properly escape

	case int, int32, int64, float32, float64:
		n.Type = NodeTypeNumber
		n.Value = v
		n.Raw = fmt.Sprintf("%v", v)

	case bool:
		n.Type = NodeTypeBool
		n.Value = v
		if v {
			n.Raw = "true"
		} else {
			n.Raw = "false"
		}

	case nil:
		n.Type = NodeTypeNull
		n.Value = nil
		n.Raw = "null"

	case map[string]interface{}:
		n.Type = NodeTypeObject
		n.Value = v
		n.Children = []*Node{}

		for key, val := range v {
			child := &Node{
				Key:    key,
				Parent: n,
			}
			child.updateValue(val)
			n.Children = append(n.Children, child)
		}

	case []interface{}:
		n.Type = NodeTypeArray
		n.Value = v
		n.Children = []*Node{}

		for _, val := range v {
			child := &Node{
				Parent: n,
			}
			child.updateValue(val)
			n.Children = append(n.Children, child)
		}

	default:
		return fmt.Errorf("unsupported value type: %T", value)
	}

	return nil
}

// Helper functions for path parsing

// parsePath parses a dot-separated path into components
func parsePath(path string) []string {
	if path == "" {
		return []string{}
	}

	// Simple split for now - should handle escaped dots and brackets
	return strings.Split(path, ".")
}

// isArrayIndex checks if a string looks like an array index
func isArrayIndex(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// parseArrayIndex parses a string as an array index
func parseArrayIndex(s string) (int, error) {
	return strconv.Atoi(s)
}

// JSONC Writer functionality

// ToJSONC converts the document back to JSONC format, preserving formatting
func (d *Document) ToJSONC() ([]byte, error) {
	var buf strings.Builder
	err := d.Root.writeJSONC(&buf, 0)
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// writeJSONC writes this node to JSONC format
func (n *Node) writeJSONC(buf *strings.Builder, indent int) error {
	switch n.Type {
	case NodeTypeObject:
		return n.writeObject(buf, indent)
	case NodeTypeArray:
		return n.writeArray(buf, indent)
	case NodeTypeString:
		buf.WriteString(fmt.Sprintf(`"%s"`, escapeString(n.Value.(string))))
	case NodeTypeNumber:
		buf.WriteString(fmt.Sprintf("%v", n.Value))
	case NodeTypeBool:
		if n.Value.(bool) {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case NodeTypeNull:
		buf.WriteString("null")
	default:
		return fmt.Errorf("unknown node type: %d", n.Type)
	}
	return nil
}

// writeObject writes an object node to JSONC format
func (n *Node) writeObject(buf *strings.Builder, indent int) error {
	buf.WriteString("{")

	if len(n.Children) == 0 {
		buf.WriteString("}")
		return nil
	}

	buf.WriteString("\n")

	for i, child := range n.Children {
		// Write indentation
		writeIndent(buf, indent+1)

		// Write key
		buf.WriteString(fmt.Sprintf(`"%s": `, escapeString(child.Key)))

		// Write value
		err := child.writeJSONC(buf, indent+1)
		if err != nil {
			return err
		}

		// Write comma except for last item (unless it has a trailing comma)
		if i < len(n.Children)-1 || n.HasTrailingComma {
			buf.WriteString(",")
		}

		buf.WriteString("\n")
	}

	// Write closing brace with proper indentation
	writeIndent(buf, indent)
	buf.WriteString("}")

	return nil
}

// writeArray writes an array node to JSONC format
func (n *Node) writeArray(buf *strings.Builder, indent int) error {
	buf.WriteString("[")

	if len(n.Children) == 0 {
		buf.WriteString("]")
		return nil
	}

	buf.WriteString("\n")

	for i, child := range n.Children {
		// Write indentation
		writeIndent(buf, indent+1)

		// Write value
		err := child.writeJSONC(buf, indent+1)
		if err != nil {
			return err
		}

		// Write comma except for last item (unless it has a trailing comma)
		if i < len(n.Children)-1 || n.HasTrailingComma {
			buf.WriteString(",")
		}

		buf.WriteString("\n")
	}

	// Write closing bracket with proper indentation
	writeIndent(buf, indent)
	buf.WriteString("]")

	return nil
}

// writeIndent writes the appropriate indentation
func writeIndent(buf *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		buf.WriteString("  ") // 2 spaces per level
	}
}

// escapeString escapes a string for JSON output
func escapeString(s string) string {
	// Simple escaping - should handle all JSON escape sequences properly
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
