package redactor

import (
	"bytes"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fariq/redact/internal/config"
)

// Redactor masks configured sensitive values in individual text lines.
type Redactor struct {
	mask          []byte
	jsonMask      []byte
	urlMask       []byte
	fields        map[string]bool
	fieldPatterns []*regexp.Regexp
	params        map[string]bool
	paramPatterns []*regexp.Regexp
}
type span struct {
	start, end, priority int
	repl                 []byte
}

// New builds a redactor from an effective rule set.
func New(e config.Effective) (*Redactor, error) {
	r := &Redactor{
		mask:     []byte(e.Mask),
		jsonMask: []byte(escapeJSON(e.Mask)),
		urlMask:  []byte(escapeURL(e.Mask)),
		fields:   make(map[string]bool, len(e.Fields)),
		params:   make(map[string]bool, len(e.URLParams)),
	}
	for _, f := range e.Fields {
		r.fields[strings.ToLower(f.Value)] = true
	}
	for _, p := range e.URLParams {
		r.params[strings.ToLower(p.Value)] = true
	}
	for _, p := range e.FieldPatterns {
		re, err := regexp.Compile("(?i:" + p.Value + ")")
		if err != nil {
			return nil, err
		}
		r.fieldPatterns = append(r.fieldPatterns, re)
	}
	for _, p := range e.URLParamPatterns {
		re, err := regexp.Compile("(?i:" + p.Value + ")")
		if err != nil {
			return nil, err
		}
		r.paramPatterns = append(r.paramPatterns, re)
	}
	return r, nil
}

// RedactLine appends line with sensitive spans masked to dst and returns the result.
func (r *Redactor) RedactLine(dst, line []byte) []byte {
	spans := r.collect(line)
	if len(spans) == 0 {
		return append(dst, line...)
	}
	spans = merge(spans)
	pos := 0
	for _, s := range spans {
		dst = append(dst, line[pos:s.start]...)
		dst = append(dst, s.repl...)
		pos = s.end
	}
	return append(dst, line[pos:]...)
}

func (r *Redactor) collect(line []byte) []span {
	var ss []span
	for i := 0; i < len(line); i++ {
		keyStart := i
		keyEnd := i
		quotedKey := false
		if line[i] == '"' || line[i] == '\'' {
			q := line[i]
			keyStart = i + 1
			keyEnd = scanQuote(line, i, q)
			quotedKey = keyEnd < len(line)
		} else {
			if !boundary(line, i) || !isFieldChar(line[i]) {
				continue
			}
			for keyEnd < len(line) && isFieldChar(line[keyEnd]) {
				keyEnd++
			}
		}
		name := strings.ToLower(string(line[keyStart:keyEnd]))
		if !r.matchField(name) {
			continue
		}
		k := keyEnd
		if quotedKey {
			k++
		}
		k = skipSpaces(line, k)
		if k >= len(line) || (line[k] != ':' && line[k] != '=') {
			continue
		}
		valStart := skipSpaces(line, k+1)
		if valStart >= len(line) {
			continue
		}
		if line[valStart] == '"' || line[valStart] == '\'' {
			q := line[valStart]
			end := scanQuote(line, valStart, q)
			repl := r.mask
			pri := 1
			if q == '"' {
				repl = r.jsonMask
				pri = 3
			}
			ss = append(ss, span{valStart + 1, end, pri, repl})
			continue
		}
		end := valueEnd(line, valStart, line[k])
		if end > valStart {
			ss = append(ss, span{valStart, end, 1, r.mask})
		}
	}
	ss = append(ss, r.urlSpans(line)...)
	return ss
}

func (r *Redactor) urlSpans(line []byte) []span {
	var ss []span
	for i := 0; i < len(line); i++ {
		if line[i] != '?' && line[i] != '&' {
			continue
		}
		nameStart := i + 1
		nameEnd := nameStart
		for nameEnd < len(line) && line[nameEnd] != '=' && line[nameEnd] != '&' && line[nameEnd] != '#' && !isSpace(line[nameEnd]) {
			nameEnd++
		}
		if nameEnd >= len(line) || line[nameEnd] != '=' {
			continue
		}
		name := strings.ToLower(string(line[nameStart:nameEnd]))
		if !r.matchParam(name) {
			continue
		}
		vs := nameEnd + 1
		ve := vs
		for ve < len(line) && line[ve] != '&' && line[ve] != '#' && !isSpace(line[ve]) && line[ve] != '"' && line[ve] != '\'' {
			ve++
		}
		ss = append(ss, span{vs, ve, 2, r.urlMask})
	}
	return ss
}

func (r *Redactor) matchField(s string) bool {
	if r.fields[s] {
		return true
	}
	for _, re := range r.fieldPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}
func (r *Redactor) matchParam(s string) bool {
	if r.params[s] {
		return true
	}
	for _, re := range r.paramPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func merge(ss []span) []span {
	sort.Slice(ss, func(i, j int) bool {
		if ss[i].start == ss[j].start {
			return ss[i].end > ss[j].end
		}
		return ss[i].start < ss[j].start
	})
	out := ss[:0]
	for _, s := range ss {
		if len(out) == 0 || s.start >= out[len(out)-1].end {
			out = append(out, s)
			continue
		}
		last := &out[len(out)-1]
		if s.end > last.end {
			last.end = s.end
		}
		if s.priority > last.priority {
			last.priority = s.priority
			last.repl = s.repl
		}
	}
	return out
}

func boundary(b []byte, i int) bool {
	return i == 0 || (!isFieldChar(b[i-1]) && b[i-1] != '?' && b[i-1] != '&')
}
func isFieldChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.'
}
func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
func skipSpaces(b []byte, i int) int {
	for i < len(b) && (b[i] == ' ' || b[i] == '\t') {
		i++
	}
	return i
}
func scanQuote(b []byte, i int, q byte) int {
	for j := i + 1; j < len(b); j++ {
		if b[j] == '\\' {
			j++
			continue
		}
		if b[j] == q {
			return j
		}
	}
	return len(b)
}
func valueEnd(b []byte, i int, sep byte) int {
	for j := i; j < len(b); j++ {
		if b[j] == '\r' || b[j] == '\n' || b[j] == ',' || b[j] == ')' || b[j] == ']' || b[j] == '}' || b[j] == '"' || b[j] == '\'' {
			return j
		}
		if sep == '=' && (b[j] == ' ' || b[j] == '\t') {
			return j
		}
		if sep == ':' && j > i && (b[j] == ' ' || b[j] == '\t') && looksKeyEq(b[j+1:]) {
			return j
		}
	}
	return len(bytes.TrimRight(b[i:], " \t\r\n")) + i
}
func looksKeyEq(b []byte) bool {
	i := 0
	for i < len(b) && isFieldChar(b[i]) {
		i++
	}
	i = skipSpaces(b, i)
	return i < len(b) && b[i] == '='
}
func escapeJSON(s string) string { q := strconv.Quote(s); return q[1 : len(q)-1] }
func escapeURL(s string) string  { return strings.ReplaceAll(url.QueryEscape(s), "%2A", "*") }
