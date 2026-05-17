package entity

import (
	"fmt"
	"reflect"

	bxerrors "bitrixgo/bitrixgo/errors"
)

// Relation описывает many-to-one связь дочерней сущности с родительской.
type Relation struct {
	Name       string
	Target     reflect.Type
	FKColumn   string
	FieldIndex int
	FKIndex    int
}

// RelationByName возвращает связь по имени rel-поля.
func (m *Meta) RelationByName(name string) (*Relation, error) {
	for i := range m.Relations {
		if m.Relations[i].Name == name {
			return &m.Relations[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", bxerrors.ErrRelationNotFound, name)
}

// RelationTo возвращает связь из meta дочерней сущности на тип родителя P.
func RelationTo[Parent any](child *Meta) (*Relation, error) {
	var zero Parent
	target := reflect.TypeOf(zero)
	if target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	for i := range child.Relations {
		if child.Relations[i].Target == target {
			return &child.Relations[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", bxerrors.ErrRelationNotFound, target.Name())
}

// DefaultRelation возвращает единственную связь или ошибку, если их несколько.
func (m *Meta) DefaultRelation() (*Relation, error) {
	if len(m.Relations) == 0 {
		return nil, fmt.Errorf("%w: no relations defined", bxerrors.ErrRelationNotFound)
	}
	if len(m.Relations) > 1 {
		return nil, fmt.Errorf("%w: multiple relations, specify name", bxerrors.ErrInvalidRelation)
	}
	return &m.Relations[0], nil
}

// ResolveRelation возвращает связь по имени или единственную по умолчанию, если имя пустое.
func (m *Meta) ResolveRelation(name string) (*Relation, error) {
	if name == "" {
		return m.DefaultRelation()
	}
	return m.RelationByName(name)
}
