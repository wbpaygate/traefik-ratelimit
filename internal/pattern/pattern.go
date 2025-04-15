package pattern

import (
	"bytes"
)

const (
	typeVal = iota
	typeAny
	typeEnd
)

type Pattern struct {
	value        []byte // /part/otherpart/*/other$
	prefix       []byte // /part
	patternParts []patternPart
}
type patternPart struct {
	pos   int
	typ   int8   // typeAny=`*`, typeEnd=`$`, typeVal=все остальное
	value []byte // `*`, `other$` или часть пути(прим.: otherpart)
}

func NewPattern(pattern string) *Pattern {
	bytesPattern := []byte(pattern)

	patternParts := bytes.Split(bytesPattern, []byte("/"))

	var pp []patternPart
	for i, part := range patternParts {
		var typ int8 = typeVal
		var newPart = part
		if bytes.Equal(part, []byte("*")) {
			typ = typeAny

		} else if bytes.HasSuffix(part, []byte("$")) {
			newPart = bytes.TrimSuffix(part, []byte("$"))
			typ = typeEnd
		}

		pp = append(pp, patternPart{
			pos:   i,
			typ:   typ,
			value: newPart,
		})
	}

	return &Pattern{
		value:        bytesPattern,
		prefix:       append([]byte("/"), patternParts[0]...),
		patternParts: pp,
	}
}

func (p *Pattern) String() string {
	return string(p.value)
}

func (p *Pattern) Match(urlPath []byte) bool {
	if !bytes.HasPrefix(urlPath, p.prefix) {
		return false
	}

	// ручной перебор, для исключения аллокация
	partStart := 0
	partNum := 0
	for i := 0; i <= len(urlPath); i++ {
		if i == len(urlPath) || urlPath[i] == '/' {
			if partNum >= len(p.patternParts) {
				return false
			}

			part := urlPath[partStart:i]
			pp := p.patternParts[partNum]

			switch pp.typ {
			case typeVal:
				if !bytes.Equal(part, pp.value) {
					return false
				}
			case typeEnd:
				if !bytes.Equal(part, pp.value) {
					return false
				}
			case typeAny:
				// * matches anything
			}

			partNum++
			partStart = i + 1
		}
	}

	return partNum == len(p.patternParts)
}
