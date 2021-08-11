package parser

// AttributeList is a double FiFo List to hold Attributes
// enables order-sensitivity when parsing Attributes
type AttributeList struct {
	keys   []string
	values []string
}

func NewAttributeList() AttributeList {
	return AttributeList{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}
}

func (l *AttributeList) Len() int { return len(l.keys) }

func (l *AttributeList) Push(key, value string) {
	if l.keys != nil {
		l.keys = append(l.keys, key)
		l.values = append(l.values, value)
	} else {
		l.keys = []string{key}
		l.values = []string{value}
	}

}

func (l *AttributeList) Pop() (string, string) {
	if l.Len() == 0 {
		return "", ""
	}
	key, val := l.keys[0], l.values[0]
	l.keys = l.keys[1:]
	l.values = l.values[1:]
	return key, val
}

func (l *AttributeList) Set(key, val string) {
	l.Push(key, val)
}

func (l *AttributeList) Merge(other AttributeList) AttributeList {
	l.keys = append(l.keys, other.keys...)
	l.values = append(l.values, other.values...)
	return *l
}

func (l *AttributeList) Has(key string) bool {
	for _, k := range l.keys {
		if k == key {
			return true
		}
	}
	return false
}
