package homework_delete

import (
	"reflect"
	"strings"
)

type Deleter[T any] struct {
	table string
	where []Predicate
	args  []any
	sb    strings.Builder
}

func (d *Deleter[T]) Build() (*Query, error) {
	d.sb.WriteString("DELETE FROM ")
	if d.table == "" {
		var t T
		d.sb.WriteByte('`')
		d.sb.WriteString(reflect.TypeOf(t).Name())
		d.sb.WriteByte('`')
	} else {
		d.sb.WriteString(d.table)
	}

	if len(d.where) > 0 {
		d.sb.WriteString(" WHERE ")
		p := d.where[0]
		for i := 1; i < len(d.where); i++ {
			p = p.And(d.where[i])
		}
		args, err := buildExpression(&d.sb, p)
		if err != nil {
			return nil, err
		}
		d.args = args
	}

	d.sb.WriteByte(';')
	return &Query{d.sb.String(), d.args}, nil
}

// From accepts model definition
func (d *Deleter[T]) From(table string) *Deleter[T] {
	d.table = table
	return d
}

// Where accepts predicates
func (d *Deleter[T]) Where(predicates ...Predicate) *Deleter[T] {
	d.where = predicates
	return d
}
