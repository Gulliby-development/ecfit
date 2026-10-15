package config

import (
	"strconv"
	"strings"
)

type globSequence struct{ tokens []globToken }
type globTokenKind uint8

const (
	globLiteral globTokenKind = iota
	globStar
	globDoubleStar
	globDirectories
	globQuestion
	globClass
	globAlternatives
	globNumberRange
)

type globToken struct {
	kind             globTokenKind
	literal          rune
	class            globCharacterClass
	alternatives     []*globSequence
	minimum, maximum int64
}
type globCharacterClass struct {
	negated bool
	ranges  []runeRange
}
type runeRange struct{ first, last rune }
type globParser struct{ pattern []rune }
type globMatchKey struct {
	sequence        *globSequence
	index, position int
}
type globMatcher struct {
	path []rune
	memo map[globMatchKey][]int
}

// Match reports whether path matches the EditorConfig glob. Alternatives stay
// as parsed branches, avoiding expansion into every possible pattern.
func Match(glob, path string) bool {
	pattern := []rune(glob)
	hasSlash := strings.ContainsRune(glob, '/')
	if hasSlash && len(pattern) > 0 && pattern[0] == '/' {
		pattern = pattern[1:]
	}
	sequence := (&globParser{pattern: pattern}).parseSequence(0, len(pattern))
	if !hasSlash {
		sequence.tokens = append([]globToken{{kind: globDirectories}}, sequence.tokens...)
	}
	matcher := globMatcher{path: []rune(path), memo: make(map[globMatchKey][]int)}
	for _, end := range matcher.matchEnds(sequence, 0, 0) {
		if end == len(matcher.path) {
			return true
		}
	}
	return false
}
func (p *globParser) parseSequence(start, end int) *globSequence {
	sequence := &globSequence{}
	for i := start; i < end; {
		switch p.pattern[i] {
		case '\\':
			literal := '\\'
			i++
			if i < end {
				literal, i = p.pattern[i], i+1
			}
			sequence.tokens = append(sequence.tokens, globToken{kind: globLiteral, literal: literal})
		case '*':
			if i+1 < end && p.pattern[i+1] == '*' {
				kind, width := globDoubleStar, 2
				if i+2 < end && p.pattern[i+2] == '/' {
					kind, width = globDirectories, 3
				}
				sequence.tokens = append(sequence.tokens, globToken{kind: kind})
				i += width
			} else {
				sequence.tokens = append(sequence.tokens, globToken{kind: globStar})
				i++
			}
		case '?':
			sequence.tokens = append(sequence.tokens, globToken{kind: globQuestion})
			i++
		case '[':
			class, next, ok := p.parseClass(i, end)
			if ok {
				sequence.tokens = append(sequence.tokens, globToken{kind: globClass, class: class})
				i = next
			} else {
				sequence.tokens = append(sequence.tokens, globToken{kind: globLiteral, literal: '['})
				i++
			}
		case '{':
			token, next, ok := p.parseBrace(i, end)
			if ok {
				sequence.tokens, i = append(sequence.tokens, token), next
			} else {
				sequence.tokens = append(sequence.tokens, globToken{kind: globLiteral, literal: '{'})
				i++
			}
		default:
			sequence.tokens = append(sequence.tokens, globToken{kind: globLiteral, literal: p.pattern[i]})
			i++
		}
	}
	return sequence
}
func (p *globParser) parseClass(start, end int) (globCharacterClass, int, bool) {
	var class globCharacterClass
	i := start + 1
	if i >= end {
		return class, start, false
	}
	if p.pattern[i] == '!' {
		class.negated = true
		i++
	}
	var members []classRune
	for i < end && p.pattern[i] != ']' {
		value, escaped := p.pattern[i], false
		if value == '\\' {
			escaped = true
			if i+1 < end {
				i++
				value = p.pattern[i]
			}
		}
		members = append(members, classRune{value: value, escaped: escaped})
		i++
	}
	if i == end || len(members) == 0 {
		return globCharacterClass{}, start, false
	}
	for member := 0; member < len(members); {
		if member+2 < len(members) && members[member+1].value == '-' && !members[member+1].escaped {
			class.ranges = append(class.ranges, runeRange{members[member].value, members[member+2].value})
			member += 3
			continue
		}
		character := members[member].value
		class.ranges = append(class.ranges, runeRange{character, character})
		member++
	}
	return class, i + 1, true
}

type classRune struct {
	value   rune
	escaped bool
}

func (p *globParser) parseBrace(start, end int) (globToken, int, bool) {
	close, separators, closed := p.scanBrace(start, end)
	if !closed {
		return globToken{}, start, false
	}
	contentStart, contentEnd := start+1, close
	if minimum, maximum, ok := parseNumericRange(p.pattern[contentStart:contentEnd]); ok {
		if minimum > maximum {
			minimum, maximum = maximum, minimum
		}
		return globToken{kind: globNumberRange, minimum: minimum, maximum: maximum}, close + 1, true
	}
	if len(separators) > 0 {
		alternatives := make([]*globSequence, 0, len(separators)+1)
		partStart := contentStart
		for _, separator := range separators {
			alternatives = append(alternatives, p.parseSequence(partStart, separator))
			partStart = separator + 1
		}
		alternatives = append(alternatives, p.parseSequence(partStart, contentEnd))
		return globToken{kind: globAlternatives, alternatives: alternatives}, close + 1, true
	}
	literal := []rune{'{'}
	for i := contentStart; i < contentEnd; i++ {
		if p.pattern[i] == '\\' && i+1 < contentEnd {
			i++
		}
		literal = append(literal, p.pattern[i])
	}
	literal = append(literal, '}')
	tokens := make([]globToken, 0, len(literal))
	for _, character := range literal {
		tokens = append(tokens, globToken{kind: globLiteral, literal: character})
	}
	return globToken{kind: globAlternatives, alternatives: []*globSequence{{tokens: tokens}}}, close + 1, true
}
func (p *globParser) scanBrace(start, end int) (int, []int, bool) {
	depth := 1
	var separators []int
	for i := start + 1; i < end; {
		switch p.pattern[i] {
		case '\\':
			i += 2
			if i > end {
				i = end
			}
		case '[':
			if _, next, ok := p.parseClass(i, end); ok {
				i = next
			} else {
				i++
			}
		case '{':
			depth++
			i++
		case '}':
			depth--
			if depth == 0 {
				return i, separators, true
			}
			i++
		case ',':
			if depth == 1 {
				separators = append(separators, i)
			}
			i++
		default:
			i++
		}
	}
	return 0, nil, false
}
func parseNumericRange(pattern []rune) (int64, int64, bool) {
	separator := -1
	for i := 0; i+1 < len(pattern); i++ {
		if pattern[i] == '.' && pattern[i+1] == '.' {
			if separator >= 0 {
				return 0, 0, false
			}
			separator, i = i, i+1
		}
	}
	if separator < 0 {
		return 0, 0, false
	}
	first, okFirst := parseRangeBound(pattern[:separator])
	last, okLast := parseRangeBound(pattern[separator+2:])
	return first, last, okFirst && okLast
}
func parseRangeBound(bound []rune) (int64, bool) {
	start := 0
	if len(bound) > 0 && bound[0] == '-' {
		start = 1
	}
	if len(bound)-start < 1 || len(bound)-start > 9 {
		return 0, false
	}
	for _, digit := range bound[start:] {
		if digit < '0' || digit > '9' {
			return 0, false
		}
	}
	value, err := strconv.ParseInt(string(bound), 10, 64)
	return value, err == nil
}
func (m *globMatcher) matchEnds(sequence *globSequence, index, position int) []int {
	if position > len(m.path) {
		return nil
	}
	key := globMatchKey{sequence, index, position}
	if ends, ok := m.memo[key]; ok {
		return ends
	}
	if index == len(sequence.tokens) {
		m.memo[key] = []int{position}
		return m.memo[key]
	}
	token := sequence.tokens[index]
	var next []int
	switch token.kind {
	case globLiteral:
		if position < len(m.path) && m.path[position] == token.literal {
			next = []int{position + 1}
		}
	case globQuestion:
		if position < len(m.path) && m.path[position] != '/' {
			next = []int{position + 1}
		}
	case globClass:
		if position < len(m.path) && m.path[position] != '/' && token.class.matches(m.path[position]) {
			next = []int{position + 1}
		}
	case globStar:
		next = m.starEnds(position, false)
	case globDoubleStar:
		next = m.starEnds(position, true)
	case globDirectories:
		next = m.directoryEnds(position)
	case globAlternatives:
		for _, alternative := range token.alternatives {
			next = appendUnique(next, m.matchEnds(alternative, 0, position)...)
		}
	case globNumberRange:
		next = m.numberRangeEnds(position, token.minimum, token.maximum)
	}
	var result []int
	for _, end := range next {
		result = appendUnique(result, m.matchEnds(sequence, index+1, end)...)
	}
	m.memo[key] = result
	return result
}
func (m *globMatcher) starEnds(position int, acrossSlash bool) []int {
	ends := make([]int, 0, len(m.path)-position+1)
	for end := position; end <= len(m.path); end++ {
		if end > position && !acrossSlash && m.path[end-1] == '/' {
			break
		}
		ends = append(ends, end)
	}
	return ends
}
func (m *globMatcher) directoryEnds(position int) []int {
	ends := []int{position}
	for start := position; start < len(m.path); {
		end := start
		for end < len(m.path) && m.path[end] != '/' {
			end++
		}
		if end == start || end == len(m.path) {
			break
		}
		ends = append(ends, end+1)
		start = end + 1
	}
	return ends
}
func (m *globMatcher) numberRangeEnds(position int, minimum, maximum int64) []int {
	var ends []int
	limit := position + 10
	if limit > len(m.path) {
		limit = len(m.path)
	}
	for end := position + 1; end <= limit; end++ {
		candidate := m.path[position:end]
		if !canonicalInteger(candidate) {
			continue
		}
		value, err := strconv.ParseInt(string(candidate), 10, 64)
		if err == nil && value >= minimum && value <= maximum {
			ends = append(ends, end)
		}
	}
	return ends
}
func canonicalInteger(number []rune) bool {
	if len(number) == 0 {
		return false
	}
	start := 0
	if number[0] == '-' {
		start = 1
		if len(number) == 1 || number[1] == '0' {
			return false
		}
	}
	if number[start] == '0' {
		return start == 0 && len(number) == 1
	}
	if number[start] < '1' || number[start] > '9' {
		return false
	}
	for _, digit := range number[start+1:] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
func appendUnique(destination []int, candidates ...int) []int {
	for _, candidate := range candidates {
		duplicate := false
		for _, existing := range destination {
			if existing == candidate {
				duplicate = true
				break
			}
		}
		if !duplicate {
			destination = append(destination, candidate)
		}
	}
	return destination
}
func (class globCharacterClass) matches(character rune) bool {
	found := false
	for _, interval := range class.ranges {
		if interval.first <= character && character <= interval.last {
			found = true
			break
		}
	}
	return found != class.negated
}
