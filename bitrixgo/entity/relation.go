package entity

import (
	"fmt"
	"reflect"

	bxerrors "bitrixgo/bitrixgo/errors"
)

// Relation описывает связь: many-to-one (struct) или one-to-many (slice + inverse).
type Relation struct {
	Name       string
	Target     reflect.Type
	FKColumn   string
	FieldIndex int
	FKIndex    int
	Inverse    bool
}

// InverseRelations возвращает one-to-many связи родителя.
func (m *Meta) InverseRelations() []Relation {
	var out []Relation
	for _, rel := range m.Relations {
		if rel.Inverse {
			out = append(out, rel)
		}
	}
	return out
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

// ChildFKRelation возвращает pseudo-relation для установки FK на дочерней сущности.
func ChildFKRelation(fkColumn string, fkIndex int) *Relation {
	return &Relation{
		FKColumn: fkColumn,
		FKIndex:  fkIndex,
	}
}

// DefaultRelation возвращает единственную many-to-one связь или ошибку, если их несколько.
func (m *Meta) DefaultRelation() (*Relation, error) {
	var m2one []Relation
	for _, rel := range m.Relations {
		if !rel.Inverse {
			m2one = append(m2one, rel)
		}
	}
	if len(m2one) == 0 {
		return nil, fmt.Errorf("%w: no relations defined", bxerrors.ErrRelationNotFound)
	}
	if len(m2one) > 1 {
		return nil, fmt.Errorf("%w: multiple relations, specify name", bxerrors.ErrInvalidRelation)
	}
	return &m2one[0], nil
}

// ResolveRelation возвращает связь по имени или единственную по умолчанию, если имя пустое.
func (m *Meta) ResolveRelation(name string) (*Relation, error) {
	if name == "" {
		return m.DefaultRelation()
	}
	return m.RelationByName(name)
}
