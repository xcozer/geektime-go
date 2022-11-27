package orm

import (
	"context"
	"gitee.com/geektime-geekbang/geektime-go/orm/internal/errs"
	"reflect"
	"strings"
)

type Selector[T any] struct {
	table string
	model *Model
	where []Predicate
	sb *strings.Builder
	args []any

	db *DB
}

// func (db *DB) NewSelector[T any]()*Selector[T] {
// 	return &Selector[T]{
// 		sb: &strings.Builder{},
// 		db: db,
// 	}
// }

func NewSelector[T any](db *DB) *Selector[T] {
	return &Selector[T]{
		sb: &strings.Builder{},
		db: db,
	}
}

// func (s *Selector[T]) Demo[S any]() (*Query, error) {
// 	panic("implement me")
// }

func (s *Selector[T]) Build() (*Query, error) {
	var err error
	s.model, err = s.db.r.Get(new(T))
	if err != nil {
		return nil, err
	}
	sb := s.sb
	sb.WriteString("SELECT * FROM ")
	// 我怎么把表名拿到
	if s.table == "" {
		sb.WriteByte('`')
		sb.WriteString(s.model.tableName)
		sb.WriteByte('`')
	} else {
		// segs := strings.Split(s.table, ".")
		// sb.WriteByte('`')
		// sb.WriteString(segs[0])
		// sb.WriteByte('`')
		// sb.WriteByte('.')
		// sb.WriteByte('`')
		// sb.WriteString(segs[1])
		// sb.WriteByte('`')
		sb.WriteString(s.table)
	}
	if len(s.where) > 0 {
		sb.WriteString(" WHERE ")
		p := s.where[0]
		for i := 1; i < len(s.where); i++ {
			p = p.And(s.where[i])
		}
		if err := s.buildExpression(p); err != nil {
			return nil, err
		}
	}

	sb.WriteByte(';')
	return &Query{
		SQL: sb.String(),
		Args: s.args,
	}, nil
}

func (s *Selector[T]) buildExpression(expr Expression) error {
	switch exp := expr.(type){
	case nil:
	case Predicate:
		// 在这里处理 p
		// p.left 构建好
		// p.op 构建好
		// p.right 构建好
		_, ok := exp.left.(Predicate)
		if ok {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.left); err != nil {
			return err
		}
		if ok {
			s.sb.WriteByte(')')
		}

		s.sb.WriteByte(' ')
		s.sb.WriteString(exp.op.String())
		s.sb.WriteByte(' ')

		_, ok = exp.right.(Predicate)
		if ok {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.right); err != nil {
			return err
		}
		if ok {
			s.sb.WriteByte(')')
		}
	case Column:
		fd, ok := s.model.fieldMap[exp.name]
		// 字段不对，或者说列不对
		if !ok {
			return errs.NewErrUnknownField(exp.name)
		}
		s.sb.WriteByte('`')
		s.sb.WriteString(fd.colName)
		s.sb.WriteByte('`')
	case value:
		s.sb.WriteByte('?')
		s.addArg(exp.val)
	default:
		return errs.NewErrUnsupportedExpression(expr)
	}
	return nil
}

func (s *Selector[T]) addArg(val any) {
	if s.args == nil {
		s.args = make([]any, 0, 8)
	}
	s.args = append(s.args, val)
}

func (s *Selector[T]) From(table string) *Selector[T] {
	s.table = table
	return s
}

// ids:=[]int{1, 2, 3,}
// s.Where("id in (?, ?, ?)", ids)
// s.Where("id in (?, ?, ?)", ids...)
// func (s *Selector[T]) Where(query string, args...any) *Selector[T] {
//
// }

func (s *Selector[T]) Where(ps...Predicate) *Selector[T] {
	s.where = ps
	return s
}

func (s *Selector[T]) Get(ctx context.Context) (*T, error) {
	q, err := s.Build()
	// 这个是构造 SQL 失败
	if err != nil {
		return nil, err
	}

	db := s.db.db
	// 在这里，就是要发起查询，并且处理结果集
	rows, err := db.QueryContext(ctx, q.SQL, q.Args...)
	// 这个是查询错误
	if err != nil {
		return nil, err
	}

	// 你要确认有没有数据
	if !rows.Next() {
		// 要不要返回 error？
		// 返回 error，和 sql 包语义保持一致
		return nil, ErrNoRows
	}

	// 在这里，继续处理结果集

	// 我怎么知道你 SELECT 出来了哪些列？
	// 拿到了 SELECT 出来的列
	cs, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 怎么利用 cs 来解决顺序问题和类型问题

	tp := new(T)


	// 通过 cs 来构造 vals
	// 怎么构造呢？
	vals := make([]any, 0, len(cs))
	valElems := make([]reflect.Value, 0, len(cs))
	for _, c := range cs {
		// c 是列名
		fd, ok := s.model.columnMap[c]
		if !ok {
			return nil, errs.NewErrUnknownColumn(c)
		}
		// 反射创建一个实例
		// 这里创建的实例是原本类型的指针类型
		// 例如 fd.Type = int，那么val 是 *int
		val := reflect.New(fd.typ)
		vals = append(vals, val.Interface())
		// 记得要调用 Elem，因为 fd.Type = int，那么val 是 *int
		valElems = append(valElems, val.Elem())
	}

	// 第一个问题：类型要匹配
	// 第二个问题：顺序要匹配

	// SELECT id, first_name, age, last_name
	// SELECT first_name, age, last_name, id
	err = rows.Scan(vals...)
	if err != nil {
		return nil, err
	}

	// 想办法把 vals 塞进去 结果 tp 里面
	tpValueElem := reflect.ValueOf(tp).Elem()
	for i, c := range cs {
		// c 是列名
		fd, ok := s.model.columnMap[c]
		if !ok {
			return nil, errs.NewErrUnknownColumn(c)
		}
		tpValueElem.FieldByName(fd.goName).
			Set(valElems[i])
	}

	return tp, nil
}

func (s *Selector[T]) GetMulti(ctx context.Context) ([]*T, error) {
	panic("implement me")
}

