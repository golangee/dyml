package util

// Attribute represents single attribute.
type Attribute struct {
	Key   string
	Value string
}

// AttributeList is a list to hold attributes.
type AttributeList struct {
	attributes []Attribute
}

// NewAttributeList creates an empty AttributeList.
func NewAttributeList() AttributeList {
	return AttributeList{}
}

// Len returns the number of attributes in the list
func (l *AttributeList) Len() int {
	return len(l.attributes)
}

// Add the attribute to the list.
func (l *AttributeList) Add(key, value string) {
	l.attributes = append(l.attributes, Attribute{
		Key:   key,
		Value: value,
	})
}

// Pop returns the *first* attribute and removes it from the list.
// Returns nil if the list is empty.
func (l *AttributeList) Pop() *Attribute {
	if l.Len() == 0 {
		return nil
	}

	a := l.attributes[0]
	l.attributes = l.attributes[1:]

	return &a
}

// Set the given attribute if it already exists or create a new
// one otherwise. Returns true if an existing attribute got overwritten.
func (l *AttributeList) Set(key, val string) bool {
	existing := l.Get(key)
	if existing != nil {
		existing.Value = val
		return true
	} else {
		l.Add(key, val)
		return false
	}
}

// Merge the current list with another list.
// Attributes in "other" will be prioritized.
func (l AttributeList) Merge(other AttributeList) AttributeList {
	result := NewAttributeList()

	for _, a := range l.attributes {
		result.Set(a.Key, a.Value)
	}

	for _, a := range other.attributes {
		result.Set(a.Key, a.Value)
	}

	return result
}

// Get returns an attribute for a given key, or nil if it does not exist.
func (l *AttributeList) Get(key string) *Attribute {
	for _, a := range l.attributes {
		if a.Key == key {
			return &a
		}
	}

	return nil
}
