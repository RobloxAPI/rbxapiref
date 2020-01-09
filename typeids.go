package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robloxapi/rbxapiref/documents"
)

type typeDecParser struct {
	i int
	s string
}

func (p *typeDecParser) eof() bool {
	return p.i >= len(p.s)
}

func (p *typeDecParser) find(f func(rune) bool) string {
	if p.eof() {
		return ""
	}
	j := p.i
	for p.i < len(p.s) {
		r, w := utf8.DecodeRuneInString(p.s[p.i:])
		if !f(r) {
			return p.s[j:p.i]
		}
		p.i += w
	}
	return p.s[j:]
}

func (p *typeDecParser) match(r rune) bool {
	if p.eof() {
		return false
	}
	if c, w := utf8.DecodeRuneInString(p.s[p.i:]); c == r {
		p.i += w
		return true
	}
	return false
}

func (p *typeDecParser) string() string {
	return p.s[p.i:]
}

func (p *typeDecParser) parseType(typ *typeDecType) bool {
	var t typeDecType
	if p.match('.') {
		// Variadic.
		if !p.match('.') || !p.match('.') {
			return false
		}
		t.Variadic = true
	}
	if t.Name = p.find(isVar); t.Name == "" {
		return false
	}
	if p.match('?') {
		t.Optional = true
	}
	*typ = t
	return true
}

func (p *typeDecParser) parseParams(params *[]typeDecField) bool {
	// '(' already parsed
	for {
		p.find(unicode.IsSpace)
		name := p.find(isVar)
		if name == "" {
			if !p.match(')') {
				return false
			}
			break
		}
		p.find(unicode.IsSpace)
		if !p.match(':') {
			return false
		}
		p.find(unicode.IsSpace)
		field := typeDecField{Name: name}
		if !p.parseType(&field.Returns) {
			return false
		}
		*params = append(*params, field)
		p.find(unicode.IsSpace)
		if p.match(',') {
			continue
		}
		if p.match(')') {
			break
		}
		return false
	}
	return true
}

type typeDecType struct {
	Name     string
	Variadic bool
	Optional bool
}

type typeDecField struct {
	Name    string
	Returns typeDecType
}

func isVar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func parseTypeDecField(s string) *typeDecField {
	var t typeDecField
	p := typeDecParser{s: s}
	if t.Name = p.find(isVar); t.Name == "" {
		return nil
	}
	p.find(unicode.IsSpace)
	if !p.match(':') {
		return nil
	}
	p.find(unicode.IsSpace)
	if !p.parseType(&t.Returns) {
		return nil
	}
	if !p.eof() {
		return nil
	}
	return &t
}

func (t typeDecField) ID() string {
	return "field-" + t.Name
}

type typeDecCtor struct {
	Struct     string
	Name       string
	Parameters []typeDecField
	Returns    []typeDecField
	First      bool
}

func parseTypeDecCtor(s string) *typeDecCtor {
	var t typeDecCtor
	p := typeDecParser{s: s}
	if t.Struct = p.find(isVar); t.Struct == "" {
		return nil
	}
	if !p.match('.') {
		return nil
	}
	method := parseTypeDecMethod(p.string())
	if method == nil {
		return nil
	}
	t.Name = method.Name
	t.Parameters = method.Parameters
	t.Returns = method.Returns
	return &t
}

func (t typeDecCtor) ID() string {
	if t.First {
		return "ctor-" + t.Name
	}
	s := make([]string, len(t.Parameters)+2)
	s[0] = "ctor"
	s[1] = t.Name
	for i, p := range t.Parameters {
		s[i+2] = p.Name
	}
	return strings.Join(s, "-")
}

type typeDecMethod struct {
	Name       string
	Parameters []typeDecField
	Returns    []typeDecField
}

func parseTypeDecMethod(s string) *typeDecMethod {
	var t typeDecMethod
	p := typeDecParser{s: s}
	if t.Name = p.find(isVar); t.Name == "" {
		return nil
	}
	p.find(unicode.IsSpace)
	if !p.match('(') {
		return nil
	}
	if !p.parseParams(&t.Parameters) {
		return nil
	}
	p.find(unicode.IsSpace)
	if p.match(':') {
		p.find(unicode.IsSpace)
		if p.match('(') {
			if !p.parseParams(&t.Returns) {
				return nil
			}
		} else {
			var param typeDecField
			if !p.parseType(&param.Returns) {
				return nil
			}
			t.Returns = []typeDecField{param}
		}
	}
	if !p.eof() {
		return nil
	}
	return &t
}

func (t typeDecMethod) ID() string {
	return "method-" + t.Name
}

type typeDecOperator struct {
	Op      string
	Struct  string
	Operand string
	Returns typeDecType
	Call    *typeDecMethod
}

func parseTypeDecOperator(s string) *typeDecOperator {
	var t typeDecOperator
	p := typeDecParser{s: s}
	switch {
	case p.match('-'):
		t.Op = "unm"
		p.find(unicode.IsSpace)
	case p.match('#'):
		t.Op = "len"
		p.find(unicode.IsSpace)
	}
	if t.Struct = p.find(isVar); t.Struct == "" {
		return nil
	}
	p.find(unicode.IsSpace)
	if t.Op != "" {
		// Handle unary operators.
		if !p.match(':') {
			return nil
		}
		if !p.parseType(&t.Returns) {
			return nil
		}
		if !p.eof() {
			return nil
		}
		return &t
	}
	switch {
	case p.match('+'):
		t.Op = "add"
	case p.match('-'):
		t.Op = "sub"
	case p.match('*'):
		t.Op = "mul"
	case p.match('/'):
		t.Op = "div"
	case p.match('%'):
		t.Op = "mod"
	case p.match('^'):
		t.Op = "pow"
	case p.match('.'):
		if !p.match('.') {
			return nil
		}
		t.Op = "concat"
	case p.match('='):
		if !p.match('=') {
			return nil
		}
		t.Op = "eq"
	case p.match('<'):
		if p.match('=') {
			t.Op = "le"
		} else {
			t.Op = "lt"
		}
	case p.match('['): // index/newindex?
	case p.match('('): // call
		t.Op = "call"
		if t.Call = parseTypeDecMethod(s); t.Call == nil {
			return nil
		}
		return &t
	}
	p.find(unicode.IsSpace)
	if t.Operand = p.find(isVar); t.Operand == "" {
		return nil
	}
	p.find(unicode.IsSpace)
	if t.Op == "" && !p.match(']') {
		return nil
	}
	p.find(unicode.IsSpace)
	switch {
	case p.match('='):
		t.Op = "newindex"
	case p.match(':'):
		if t.Op == "" {
			t.Op = "index"
		}
	}
	p.find(unicode.IsSpace)
	if !p.parseType(&t.Returns) {
		return nil
	}
	if !p.eof() {
		return nil
	}
	return &t
}

func (t typeDecOperator) ID() string {
	switch t.Op {
	case "unm", "len", "index", "newindex", "call":
		return "op-" + t.Op
	}
	return "op-" + t.Op + "-" + t.Operand
}

// GenerateDocumentTypeIDs scans a document for sections that indicate the
// documentation of a type entity, then ID heading IDs that follow a
// standard format.
func GenerateDocumentTypeIDs(document Document) {
	if sec := document.Query("Constructors"); sec != nil {
		subs := sec.Subsections()
		ctors := make([]*typeDecCtor, len(subs))
		firsts := map[string]*typeDecCtor{}
		for i, sub := range subs {
			ctor := parseTypeDecCtor(strings.TrimSpace(sub.Name()))
			ctors[i] = ctor
			if ctor != nil && (len(ctor.Parameters) == 0 || firsts[ctor.Name] == nil) {
				ctor.First = true
				if c, ok := firsts[ctor.Name]; ok {
					c.First = false
				}
				firsts[ctor.Name] = ctor
			}
		}
		for i, sub := range subs {
			if sub, ok := sub.(documents.Headingable); ok {
				if ctors[i] != nil {
					sub.SetHeadingID(ctors[i].ID())
				}
			}
		}
	}
	if sec := document.Query("Fields"); sec != nil {
		for _, sub := range sec.Subsections() {
			if sub, ok := sub.(documents.Headingable); ok {
				dec := parseTypeDecField(strings.TrimSpace(sub.Name()))
				if dec != nil {
					sub.SetHeadingID(dec.ID())
				}
			}
		}
	}
	if sec := document.Query("Methods"); sec != nil {
		for _, sub := range sec.Subsections() {
			if sub, ok := sub.(documents.Headingable); ok {
				dec := parseTypeDecMethod(strings.TrimSpace(sub.Name()))
				if dec != nil {
					sub.SetHeadingID(dec.ID())
				}
			}
		}
	}
	if sec := document.Query("Operators"); sec != nil {
		for _, sub := range sec.Subsections() {
			if sub, ok := sub.(documents.Headingable); ok {
				dec := parseTypeDecOperator(strings.TrimSpace(sub.Name()))
				if dec != nil {
					sub.SetHeadingID(dec.ID())
				}
			}
		}
	}
}
