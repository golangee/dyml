package parser

// AttributeList is a double FiFo List to hold Attributes
// enables order-sensitivity when parsing Attributes
// TODO: possible refactor to linked list, or other more performant datastructure

type Attribute struct {
	Key   string
	Value string
	Next  *Attribute
}

type AttributeList struct {
	first  *Attribute
	last   *Attribute
	length int
}

func NewAttributeList() AttributeList {
	return AttributeList{}
}

func (l *AttributeList) Len() int { return l.length }

func (l *AttributeList) Push(key, value *string) {
	attribute := &Attribute{
		Key:   *key,
		Value: *value,
	}

	if l.first == nil {
		l.first = attribute
		l.last = l.first
		l.length++
		return
	}

	cache := l.last
	l.last = attribute
	cache.Next = l.last
	l.length++
}

func (l *AttributeList) Pop() (*string, *string) {
	if l.Len() == 0 {
		return nil, nil
	}
	key, val := l.first.Key, l.first.Value
	if l.first == l.last {
		l.last = nil
	}
	l.first = l.first.Next
	l.length--
	return &key, &val
}

func (l *AttributeList) Set(key, val *string) {
	l.Push(key, val)
}

func (l *AttributeList) Merge(other AttributeList) AttributeList {
	if l.Len() == 0 {
		return other
	}
	if other.Len() == 0 {
		return *l
	}
	l.last.Next = other.first
	l.last = other.last
	l.length += other.Len()
	return *l
}

func (l *AttributeList) Has(key string) bool {
	if l.Len() == 0 {
		return false
	}
	attribute := l.first
	if attribute.Key == key {
		return true
	}
	for a := attribute.Next; a != nil; a = a.Next {
		if a.Key == key {
			return true
		}
	}
	return false
}

func (l *AttributeList) Get(index int) (*string, *string) {
	if l.Len() == 0 || l.Len() < index {
		return nil, nil
	}
	runner := l.first
	for i := 0; i < index; i++ {
		runner = runner.Next
	}
	return &runner.Key, &runner.Value
}
