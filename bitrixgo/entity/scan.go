package entity

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
)

// ScanRow считывает одну строку sql.Rows в dest (должен быть указателем на struct).
func ScanRow(rows *sql.Rows, meta *Meta, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return err
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("entity: dest must be non-nil pointer")
	}
	rv = rv.Elem()

	for i, col := range columns {
		f, ok := meta.FieldByColumn(col)
		if !ok {
			continue
		}
		if err := setFieldValue(rv.Field(f.Index), values[i], *f); err != nil {
			return fmt.Errorf("entity: scan column %s: %w", col, err)
		}
	}
	return nil
}

// ValuesFromEntity извлекает значения колонок из сущности для INSERT/UPDATE.
func ValuesFromEntity(meta *Meta, entity any, columns []Field) ([]any, error) {
	rv := reflect.ValueOf(entity)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity: expected struct, got %s", rv.Kind())
	}

	values := make([]any, len(columns))
	for i, f := range columns {
		v, err := fieldToDB(rv.Field(f.Index), f)
		if err != nil {
			return nil, fmt.Errorf("entity: column %s: %w", f.Column, err)
		}
		values[i] = v
	}
	return values, nil
}

// ValuesFromMap извлекает значения из map обновления, ключи — имена колонок.
func ValuesFromMap(meta *Meta, fields map[string]any) ([]string, []any, error) {
	cols := make([]string, 0, len(fields))
	vals := make([]any, 0, len(fields))
	for col, v := range fields {
		f, ok := meta.FieldByColumn(col)
		if !ok {
			cols = append(cols, col)
			vals = append(vals, v)
			continue
		}
		rv := reflect.ValueOf(v)
		converted, err := convertToField(rv, *f)
		if err != nil {
			return nil, nil, fmt.Errorf("entity: column %s: %w", col, err)
		}
		cols = append(cols, col)
		vals = append(vals, converted)
	}
	return cols, vals, nil
}

// FieldToDB преобразует значение поля struct в значение для SQL.
func FieldToDB(fv reflect.Value, f Field) (any, error) {
	return fieldToDB(fv, f)
}

func fieldToDB(fv reflect.Value, f Field) (any, error) {
	if !fv.IsValid() {
		return nil, nil
	}
	if f.BoolYN {
		return boolToYN(fv), nil
	}
	switch fv.Kind() {
	case reflect.Pointer:
		if fv.IsNil() {
			return nil, nil
		}
		return fieldToDB(fv.Elem(), f)
	case reflect.Bool:
		return boolToInt(fv), nil
	default:
		return fv.Interface(), nil
	}
}

func setFieldValue(fv reflect.Value, raw any, f Field) error {
	if raw == nil {
		return nil
	}

	if f.BoolYN {
		return setBoolYN(fv, raw)
	}

	switch fv.Kind() {
	case reflect.Pointer:
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return setFieldValue(fv.Elem(), raw, f)
	case reflect.String:
		switch v := raw.(type) {
		case string:
			fv.SetString(v)
		case []byte:
			fv.SetString(string(v))
		default:
			fv.SetString(fmt.Sprint(v))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := toInt64(raw)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := toInt64(raw)
		if err != nil {
			return err
		}
		fv.SetUint(uint64(n))
	case reflect.Float32, reflect.Float64:
		n, err := toFloat64(raw)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	case reflect.Bool:
		b, err := toBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Struct:
		if fv.Type() == reflect.TypeOf(time.Time{}) {
			switch v := raw.(type) {
			case time.Time:
				fv.Set(reflect.ValueOf(v))
			case []byte:
				t, err := time.Parse("2006-01-02 15:04:05", string(v))
				if err != nil {
					return err
				}
				fv.Set(reflect.ValueOf(t))
			case string:
				t, err := time.Parse("2006-01-02 15:04:05", v)
				if err != nil {
					return err
				}
				fv.Set(reflect.ValueOf(t))
			default:
				return fmt.Errorf("unsupported time type %T", raw)
			}
		}
	}
	return nil
}

func convertToField(rv reflect.Value, f Field) (any, error) {
	if f.BoolYN {
		if rv.Kind() == reflect.Bool {
			return boolToYN(rv), nil
		}
		if rv.Kind() == reflect.String {
			return rv.String(), nil
		}
	}
	if f.Type.Kind() == reflect.Bool {
		switch rv.Kind() {
		case reflect.Bool:
			return boolToInt(rv), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int(), nil
		}
	}
	if !rv.IsValid() {
		return nil, nil
	}
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return nil, nil
	}
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	return rv.Interface(), nil
}

func boolToInt(v reflect.Value) int64 {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Bool && v.Bool() {
		return 1
	}
	return 0
}

func boolToYN(v reflect.Value) string {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "N"
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Bool && v.Bool() {
		return "Y"
	}
	if v.Kind() == reflect.String && (v.String() == "Y" || v.String() == "y" || v.String() == "1") {
		return "Y"
	}
	return "N"
}

func setBoolYN(fv reflect.Value, raw any) error {
	b, err := ynToBool(raw)
	if err != nil {
		return err
	}
	switch fv.Kind() {
	case reflect.Bool:
		fv.SetBool(b)
	case reflect.Pointer:
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return setBoolYN(fv.Elem(), raw)
	case reflect.String:
		if b {
			fv.SetString("Y")
		} else {
			fv.SetString("N")
		}
	}
	return nil
}

func ynToBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		return v == "Y" || v == "y" || v == "1", nil
	case []byte:
		s := string(v)
		return s == "Y" || s == "y" || s == "1", nil
	case int64:
		return v != 0, nil
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("unsupported bool value %T", raw)
	}
}

func toInt64(raw any) (int64, error) {
	switch v := raw.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case []byte:
		var n int64
		_, err := fmt.Sscan(string(v), &n)
		return n, err
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", raw)
	}
}

func toFloat64(raw any) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case []byte:
		var n float64
		_, err := fmt.Sscan(string(v), &n)
		return n, err
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", raw)
	}
}

func toBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case int64:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case uint8:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case []byte:
		return dbToBool(string(v))
	case string:
		return dbToBool(v)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", raw)
	}
}

func dbToBool(s string) (bool, error) {
	switch s {
	case "1", "Y", "y":
		return true, nil
	case "0", "N", "n", "":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported bool value %q", s)
	}
}

// PrimaryKeyValue извлекает значение первичного ключа из сущности.
func PrimaryKeyValue(meta *Meta, entity any) (any, error) {
	f, ok := meta.FieldByColumn(meta.PrimaryKey)
	if !ok {
		return nil, fmt.Errorf("entity: primary key column %s not found", meta.PrimaryKey)
	}
	rv := reflect.ValueOf(entity)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	return fieldToDB(rv.Field(f.Index), *f)
}
