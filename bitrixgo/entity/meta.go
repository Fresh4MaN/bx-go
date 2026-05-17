package entity

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	bxerrors "bitrixgo/bitrixgo/errors"
)

// TableNamer реализуют сущности, задающие логическое имя таблицы.
type TableNamer interface {
	TableName() string
}

// Field описывает сопоставленное поле структуры.
type Field struct {
	Name      string
	Column    string
	Index     int
	Primary   bool
	Auto      bool
	Kind      reflect.Kind
	Type      reflect.Type
	BoolYN    bool
	Relation  bool
	RefTarget string
}

// Meta хранит метаданные сущности, полученные из struct tags.
type Meta struct {
	Type       reflect.Type
	Table      string
	PrimaryKey string
	Fields     []Field
	Relations  []Relation
	byColumn   map[string]*Field
}

var cache sync.Map

// For возвращает закэшированные метаданные для типа T.
func For[T any]() (*Meta, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if v, ok := cache.Load(t); ok {
		return v.(*Meta), nil
	}
	m, err := parseMeta(t)
	if err != nil {
		return nil, err
	}
	actual, _ := cache.LoadOrStore(t, m)
	return actual.(*Meta), nil
}

func parseMeta(t reflect.Type) (*Meta, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity: %s is not a struct", t.Name())
	}

	m := &Meta{
		Type:       t,
		PrimaryKey: "ID",
		byColumn:   make(map[string]*Field),
	}

	if table := tableFromType(t); table != "" {
		m.Table = table
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := sf.Tag.Get("bx")
		if tag == "" || tag == "-" {
			continue
		}

		if strings.HasPrefix(tag, "rel=") {
			rel, err := parseRelationField(tag, sf, i)
			if err != nil {
				return nil, fmt.Errorf("entity: %s.%s: %w", t.Name(), sf.Name, err)
			}
			m.Relations = append(m.Relations, rel)
			m.Fields = append(m.Fields, Field{
				Name:     sf.Name,
				Index:    i,
				Kind:     sf.Type.Kind(),
				Type:     sf.Type,
				Relation: true,
			})
			continue
		}

		parts := strings.Split(tag, ";")
		column := strings.TrimSpace(parts[0])
		if column == "" {
			column = sf.Name
		}

		f := Field{
			Name:   sf.Name,
			Column: column,
			Index:  i,
			Kind:   sf.Type.Kind(),
			Type:   sf.Type,
		}

		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			switch {
			case p == "pk":
				f.Primary = true
				m.PrimaryKey = column
			case p == "auto":
				f.Auto = true
			case p == "boolyn":
				f.BoolYN = true
			case strings.HasPrefix(p, "table="):
				m.Table = strings.TrimPrefix(p, "table=")
			case strings.HasPrefix(p, "ref="):
				f.RefTarget = strings.TrimPrefix(p, "ref=")
			}
		}

		m.Fields = append(m.Fields, f)
		m.byColumn[column] = &m.Fields[len(m.Fields)-1]
	}

	if err := m.validateRelations(); err != nil {
		return nil, err
	}

	if m.Table == "" {
		return nil, fmt.Errorf("entity: %s: table name required (TableName() or bx:\"table=...\")", t.Name())
	}

	if len(m.columnFields()) == 0 {
		return nil, fmt.Errorf("entity: %s: no bx-tagged fields", t.Name())
	}

	return m, nil
}

func parseRelationField(tag string, sf reflect.StructField, index int) (Relation, error) {
	target := sf.Type
	if target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if target.Kind() != reflect.Struct {
		return Relation{}, fmt.Errorf("%w: rel field must be struct", bxerrors.ErrInvalidRelation)
	}

	parts := strings.Split(tag, ";")
	targetName := strings.TrimPrefix(strings.TrimSpace(parts[0]), "rel=")
	if targetName == "" {
		targetName = target.Name()
	}

	fkColumn := ""
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "fk=") {
			fkColumn = strings.TrimPrefix(p, "fk=")
		}
	}
	if fkColumn == "" {
		return Relation{}, fmt.Errorf("%w: rel tag requires fk=", bxerrors.ErrInvalidRelation)
	}

	return Relation{
		Name:       sf.Name,
		Target:     target,
		FKColumn:   fkColumn,
		FieldIndex: index,
	}, nil
}

func (m *Meta) validateRelations() error {
	for i := range m.Relations {
		rel := &m.Relations[i]
		fk, ok := m.FieldByColumn(rel.FKColumn)
		if !ok {
			return fmt.Errorf("entity: %s: %w: unknown fk column %s", m.Type.Name(), bxerrors.ErrInvalidRelation, rel.FKColumn)
		}
		rel.FKIndex = fk.Index

		if _, err := ForType(rel.Target); err != nil {
			return fmt.Errorf("entity: %s: relation %s: %w", m.Type.Name(), rel.Name, err)
		}
	}
	return nil
}

// ForType возвращает метаданные для reflect.Type (должен быть struct).
func ForType(t reflect.Type) (*Meta, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if v, ok := cache.Load(t); ok {
		return v.(*Meta), nil
	}
	m, err := parseMeta(t)
	if err != nil {
		return nil, err
	}
	actual, _ := cache.LoadOrStore(t, m)
	return actual.(*Meta), nil
}

func tableFromType(t reflect.Type) string {
	var tn TableNamer
	iface := reflect.TypeOf(&tn).Elem()

	if reflect.PointerTo(t).Implements(iface) {
		v := reflect.New(t).Interface().(TableNamer)
		return v.TableName()
	}
	if t.Implements(iface) {
		v := reflect.New(t).Elem().Interface().(TableNamer)
		return v.TableName()
	}
	return ""
}

func (m *Meta) columnFields() []Field {
	out := make([]Field, 0, len(m.Fields))
	for _, f := range m.Fields {
		if !f.Relation {
			out = append(out, f)
		}
	}
	return out
}

// FieldByColumn возвращает метаданные поля по имени колонки.
func (m *Meta) FieldByColumn(column string) (*Field, bool) {
	f, ok := m.byColumn[column]
	return f, ok
}

// FieldByName возвращает метаданные поля по имени поля структуры.
func (m *Meta) FieldByName(name string) (*Field, bool) {
	for i := range m.Fields {
		if m.Fields[i].Name == name {
			return &m.Fields[i], true
		}
	}
	return nil, false
}

// InsertColumns возвращает колонки и поля для INSERT (без pk+auto и rel-полей).
func (m *Meta) InsertColumns() []Field {
	out := make([]Field, 0, len(m.Fields))
	for _, f := range m.Fields {
		if f.Relation {
			continue
		}
		if f.Primary && f.Auto {
			continue
		}
		out = append(out, f)
	}
	return out
}

// SelectColumns возвращает все сопоставленные имена колонок (без rel-полей).
func (m *Meta) SelectColumns() []string {
	cols := make([]Field, 0, len(m.Fields))
	for _, f := range m.Fields {
		if f.Relation {
			continue
		}
		cols = append(cols, f)
	}
	names := make([]string, len(cols))
	for i, f := range cols {
		names[i] = f.Column
	}
	return names
}
