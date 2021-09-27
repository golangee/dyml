package parser

// Attribute represents single attribute and holds a pointer to the next attribute
type Attribute struct {
	Key   string
	Value string
	Next  *Attribute
}

// AttributeList is a FiFo linked list to hold Attributes
// enables order-sensitivity when parsing Attributes
type AttributeList struct {
	first  *Attribute
	last   *Attribute
	length int
}

// NewAttributeList creates an empty AttributeList
func NewAttributeList() AttributeList {
	return AttributeList{}
}

// Len returns the number of attributes in the list
func (l *AttributeList) Len() int { return l.length }

// Push adds an attribute to the list
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

// Pop returns the first attribute and removes it from the list.
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

// Set = Pop
func (l *AttributeList) Set(key, val *string) {
	l.Push(key, val)
}

// Merge merges the AttributeList with another given AttributeList.
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

// Has returns true if the given key exists in the AttributeList.
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

// Get returns the key and value of the attribute on the given position in the AttributeList.
// returns (nil, nil) if the index is out of bounds
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

func (l *AttributeList) GetKey(key string) *string {
	for a := l.first; a != nil; a = a.Next {
		if a.Key == key {
			return &a.Value
		}
	}

	return nil
}