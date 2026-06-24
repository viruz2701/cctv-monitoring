// Package workflow — Condition evaluator (WF-9.1.2).
//
// Built-in evaluator with CEL-ready interface.
// В будущем: заменить на cel-go (github.com/google/cel-go, Apache 2.0).
package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// EvalContext — контекст для evaluation условий.
type EvalContext map[string]interface{}

// Evaluate проверяет, удовлетворяет ли контекст условию.
func Evaluate(cond WorkflowCondition, ctx EvalContext) (bool, error) {
	// Получаем значение поля из контекста (поддержка dot notation)
	actual, err := resolveField(cond.Field, ctx)
	if err != nil {
		return false, fmt.Errorf("resolve field %q: %w", cond.Field, err)
	}

	return compare(actual, cond.Operator, cond.Value)
}

// EvaluateAll проверяет ВСЕ условия (AND).
func EvaluateAll(conditions []WorkflowCondition, ctx EvalContext) (bool, error) {
	for _, cond := range conditions {
		ok, err := Evaluate(cond, ctx)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// resolveField извлекает значение из контекста по dot notation.
// Примеры: "event.severity", "device.name", "part.stock"
func resolveField(field string, ctx EvalContext) (interface{}, error) {
	parts := strings.Split(field, ".")
	current := interface{}(ctx)

	for _, part := range parts {
		switch v := current.(type) {
		case EvalContext:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("field %q not found", field)
			}
			current = val
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("field %q not found", field)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot traverse %q: %T is not a map", field, current)
		}
	}

	return current, nil
}

// compare сравнивает два значения с заданным оператором.
func compare(actual interface{}, op ConditionOp, expected interface{}) (bool, error) {
	switch op {
	case OpEQ:
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected), nil
	case OpNEQ:
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected), nil
	case OpGT, OpGTE, OpLT, OpLTE:
		return compareNumeric(actual, op, expected)
	case OpContains:
		return compareContains(actual, expected), nil
	case OpMatches:
		return compareMatches(actual, expected)
	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

func compareNumeric(actual interface{}, op ConditionOp, expected interface{}) (bool, error) {
	a, err := toFloat64(actual)
	if err != nil {
		return false, fmt.Errorf("cannot convert actual %v to number: %w", actual, err)
	}
	e, err := toFloat64(expected)
	if err != nil {
		return false, fmt.Errorf("cannot convert expected %v to number: %w", expected, err)
	}

	switch op {
	case OpGT:
		return a > e, nil
	case OpGTE:
		return a >= e, nil
	case OpLT:
		return a < e, nil
	case OpLTE:
		return a <= e, nil
	}
	return false, nil
}

func compareContains(actual, expected interface{}) bool {
	switch v := actual.(type) {
	case string:
		return strings.Contains(v, fmt.Sprintf("%v", expected))
	case []interface{}:
		for _, item := range v {
			if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", expected) {
				return true
			}
		}
	}
	return false
}

func compareMatches(actual, expected interface{}) (bool, error) {
	s, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("matches requires string, got %T", actual)
	}
	pattern, ok := expected.(string)
	if !ok {
		return false, fmt.Errorf("matches pattern must be string, got %T", expected)
	}
	return regexp.MatchString(pattern, s)
}

func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

// FillTemplate заполняет шаблон значениями из контекста.
// Поддерживает {field.nested.field} синтаксис.
func FillTemplate(tpl string, ctx EvalContext) string {
	result := tpl
	for {
		start := strings.Index(result, "{")
		if start < 0 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end < 0 {
			break
		}
		key := result[start+1 : start+end]
		val, err := resolveField(key, ctx)
		if err != nil {
			result = result[:start] + "<unknown>" + result[start+end+1:]
		} else {
			result = result[:start] + fmt.Sprintf("%v", val) + result[start+end+1:]
		}
	}
	return result
}
