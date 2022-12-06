package homework_delete

import (
	"fmt"
	"strings"
)

func buildExpression(sb *strings.Builder, expr Expression) (args []any, err error) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case Column:
		sb.WriteByte('`')
		sb.WriteString(e.name)
		sb.WriteByte('`')
	case value:
		sb.WriteByte('?')
		args = append(args, e.val)
	case Predicate:
		_, isLeftPredicate := e.left.(Predicate)
		if isLeftPredicate {
			sb.WriteByte('(')
		}
		lArgs, err := buildExpression(sb, e.left)
		if err != nil {
			return nil, err
		}
		if isLeftPredicate {
			sb.WriteByte(')')
		}
		args = append(args, lArgs...)

		sb.WriteByte(' ')
		sb.WriteString(e.op.String())
		sb.WriteByte(' ')

		_, isRightPredicate := e.right.(Predicate)
		if isRightPredicate {
			sb.WriteByte('(')
		}
		rArgs, err := buildExpression(sb, e.right)
		if err != nil {
			return nil, err
		}
		if isRightPredicate {
			sb.WriteByte(')')
		}
		args = append(args, rArgs...)
	default:
		return nil, fmt.Errorf("orm: 不支持的表达式 %v", e)
	}
	return
}
