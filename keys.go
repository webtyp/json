package json

import (
	"unsafe"

	"webtyp.com/fmt"
)

// Keys returns the names of a JSON object's fields, in the order they appear: for reading an
// object whose field names are data, such as the "properties" of a JSON Schema. Decode reads
// fields whose names the code already knows.
func Keys(object string) ([]string, error) {
	p := parser{data: unsafe.Slice(unsafe.StringData(object), len(object))}
	p.skipWhitespace()
	if p.next() != '{' {
		return nil, fmt.Err("json", "keys", "expected object")
	}
	var keys []string
	p.skipWhitespace()
	if p.peek() == '}' {
		return keys, nil
	}
	for {
		p.skipWhitespace()
		if p.next() != '"' {
			return nil, fmt.Err("json", "keys", "expected quote")
		}
		key, err := p.parseString()
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		p.skipWhitespace()
		if p.next() != ':' {
			return nil, fmt.Err("json", "keys", "expected :")
		}
		if err := p.skipValue(); err != nil {
			return nil, err
		}
		p.skipWhitespace()
		switch p.next() {
		case '}':
			return keys, nil
		case ',':
		default:
			return nil, fmt.Err("json", "keys", "expected , or }")
		}
	}
}
