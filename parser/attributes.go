package parser

// AttributeList is a double FiFo List to hold Attributes
// enables order-sensitivity when parsing Attributes
type AttributeList struct {
	Keys   []*string
	Values []*string
}

func NewAttributeList() AttributeList {
	return AttributeList{}
}

func (l *AttributeList) Len() int { return len(l.Keys) }

func (l *AttributeList) Push(key, value *string) {
	if l.Keys != nil {
		l.Keys = append(l.Keys, key)
		l.Values = append(l.Values, value)
	} else {
		l.Keys = []*string{key}
		l.Values = []*string{value}
	}
}

func (l *AttributeList) Pop() (*string, *string) {
	if l.Len() == 0 {
		return nil, nil
	}
	key, val := l.Keys[0], l.Values[0]
	l.Keys = l.Keys[1:]
	l.Values = l.Values[1:]
	return key, val
}

func (l *AttributeList) Set(key, val *string) {
	l.Push(key, val)
}

func (l *AttributeList) Merge(other AttributeList) AttributeList {
	l.Keys = append(l.Keys, other.Keys...)
	l.Values = append(l.Values, other.Values...)
	return *l
}

func (l *AttributeList) Has(key string) bool {
	for _, k := range l.Keys {
		if *k == key {
			return true
		}
	}
	return false
}

func (l *AttributeList) Get(index int) (*string, *string) {
	if l.Len() == 0 {
		return nil, nil
	}
	return l.Values[index], l.Keys[index]
}
