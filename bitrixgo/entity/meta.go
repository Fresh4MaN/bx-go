package entity

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// TableNamer реализуют сущности, задающие логическое имя таблицы.
type TableNamer interface {
	TableName() string
}

// Field описывает сопоставленное поле структуры.
type Field struct {
	Name     string
	Column   string
	Index    int
	Primary  bool
	Auto     bool
	Kind     reflect.Kind
	Type     reflect.Type
	BoolYN   bool
}

// Meta хранит метаданные сущности, полученные из struct tags.
type Meta struct {
	Type       reflect.Type
	Table      string
	PrimaryKey string
	Fields     []Field
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
			}
		}

		m.Fields = append(m.Fields, f)
		m.byColumn[column] = &m.Fields[len(m.Fields)-1]
	}

	if m.Table == "" {
		return nil, fmt.Errorf("entity: %s: table name required (TableName() or bx:\"table=...\")", t.Name())
	}

	if len(m.Fields) == 0 {
		return nil, fmt.Errorf("entity: %s: no bx-tagged fields", t.Name())
	}

	return m, nil
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

// FieldByColumn возвращает метаданные поля по имени колонки.
func (m *Meta) FieldByColumn(column string) (*Field, bool) {
	f, ok := m.byColumn[column]
	return f, ok
}

// InsertColumns возвращает колонки и поля для INSERT (без pk+auto).
func (m *Meta) InsertColumns() []Field {
	out := make([]Field, 0, len(m.Fields))
	for _, f := range m.Fields {
		if f.Primary && f.Auto {
			continue
		}
		out = append(out, f)
	}
	return out
}

// SelectColumns возвращает все сопоставленные имена колонок.
func (m *Meta) SelectColumns() []string {
	cols := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		cols[i] = f.Column
	}
	return cols
}
