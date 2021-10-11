package util

import "github.com/golangee/dyml/token"

// Attribute represents single attribute.
type Attribute struct {
	Key   string
	Value string
	Range token.Position
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

// Add the given attribute to the list.
func (l *AttributeList) Add(attr Attribute) {
	l.attributes = append(l.attributes, attr)
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
func (l *AttributeList) Set(attr Attribute) bool {
	existing := l.Get(attr.Key)
	if existing != nil {
		existing = &attr
		return true
	} else {
		l.Add(attr)
		return false
	}
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
