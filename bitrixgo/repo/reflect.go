package repo

import "reflect"

func reflectValueOf[T any](item *T) reflect.Value {
	return reflect.ValueOf(item).Elem()
}
