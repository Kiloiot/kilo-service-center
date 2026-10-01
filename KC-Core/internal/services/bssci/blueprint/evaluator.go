// Package blueprint provides expression evaluation for blueprint func and condition fields.
package blueprint

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/numconv"
)

// ExpressionEvaluator handles func and condition expression evaluation for blueprints.
type ExpressionEvaluator struct {
	// specFieldRefPattern matches $<name> references per MIOTY Application Layer Spec Table 15
	specFieldRefPattern *regexp.Regexp
	// calibrationRefPattern matches $calibration.key references
	calibrationRefPattern *regexp.Regexp
}

const (
	specFieldRefRegex   = `\$([a-zA-Z]\w*)`
	calibrationRefRegex = `\$calibration\.(\w+)`

	tokenValueRef      = "$value"
	keywordValue       = "value"
	keywordField       = "field"
	keywordCalibration = "calibration"
	literalTrue        = "true"
	literalFalse       = "false"

	missingCalibrationValue = "1"
	missingFieldValue       = "0"

	// growSlack pre-sizes the normalization builder for a few bare-$
	// expansions.
	growSlack = 16
)

// Math function names the expression grammar accepts.
const (
	fnSqrt = "sqrt"
	fnAbs  = "abs"
	fnLog  = "log"
	fnExp  = "exp"
	fnSin  = "sin"
	fnCos  = "cos"
	fnTan  = "tan"
)

// mathFunctions lists the accepted function names in parse order.
var mathFunctions = []string{fnSqrt, fnAbs, fnLog, fnExp, fnSin, fnCos, fnTan}

// NewExpressionEvaluator creates a new expression evaluator.
// Blueprint expression grammar (MIOTY Application Layer Spec Table 15):
// reference tokens, keyword names excluded from field substitution, boolean
// literals, and the defaults substituted for missing references.
func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{
		specFieldRefPattern:   regexp.MustCompile(specFieldRefRegex),
		calibrationRefPattern: regexp.MustCompile(calibrationRefRegex),
	}
}

// EvaluateCondition evaluates a condition expression and returns true if the condition is met.
// Condition expressions support simple comparisons like "$field.type == 1" or "$field.enabled == true".
func (e *ExpressionEvaluator) EvaluateCondition(
	condition string,
	fields map[string]interface{},
	calibration map[string]interface{},
) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
	}

	// Normalize spec syntax: bare $ → $value (per MIOTY Application Layer Spec Table 15)
	condition = e.normalizeBareCurrentValue(condition)

	// Substitute field and calibration references
	condition = e.substituteCalibrationRefs(condition, calibration)
	condition = e.substituteSpecFieldRefs(condition, fields)

	// Parse and evaluate simple conditions
	// Support: ==, !=, <, >, <=, >=, &&, ||
	return e.evaluateBooleanExpression(condition)
}

// EvaluateFunc evaluates a func expression and returns the computed value.
// Func expressions support arithmetic operations and references to other fields/calibration.
// Examples:
//   - "$value * 0.1 + 273.15" (convert raw to Kelvin)
//   - "$value * $calibration.factor" (apply calibration factor)
//   - "$field.temperature + $field.offset" (combine fields)
func (e *ExpressionEvaluator) EvaluateFunc(
	funcExpr string,
	currentValue interface{},
	fields map[string]interface{},
	calibration map[string]interface{},
) (interface{}, error) {
	funcExpr = strings.TrimSpace(funcExpr)
	if funcExpr == "" {
		return currentValue, nil
	}

	// Normalize spec syntax: bare $ → $value (per MIOTY Application Layer Spec Table 15)
	funcExpr = e.normalizeBareCurrentValue(funcExpr)

	// Replace $value with the current value
	if numVal, ok := numconv.ToFloat64(currentValue); ok {
		funcExpr = strings.ReplaceAll(funcExpr, tokenValueRef, fmt.Sprintf("%v", numVal))
	} else {
		funcExpr = strings.ReplaceAll(funcExpr, tokenValueRef, fmt.Sprintf("%v", currentValue))
	}

	// Substitute field and calibration references
	funcExpr = e.substituteCalibrationRefs(funcExpr, calibration)
	funcExpr = e.substituteSpecFieldRefs(funcExpr, fields)

	// Evaluate arithmetic expression
	return e.evaluateArithmeticExpression(funcExpr)
}

// GetCalibrationValue retrieves a calibration value by key.
func (e *ExpressionEvaluator) GetCalibrationValue(key string, calibration map[string]interface{}) (interface{}, error) {
	if calibration == nil {
		return nil, errCalibrationNotProvided
	}
	val, ok := calibration[key]
	if !ok {
		return nil, fmt.Errorf(errFmtCalibrationKeyNotFound, key)
	}
	return val, nil
}

// substituteCalibrationRefs replaces $calibration.key references with their values.
func (e *ExpressionEvaluator) substituteCalibrationRefs(expr string, calibration map[string]interface{}) string {
	return e.calibrationRefPattern.ReplaceAllStringFunc(expr, func(match string) string {
		matches := e.calibrationRefPattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		key := matches[1]
		if val, ok := calibration[key]; ok {
			if numVal, isNum := numconv.ToFloat64(val); isNum {
				return fmt.Sprintf("%v", numVal)
			}
			return fmt.Sprintf("%v", val)
		}
		return missingCalibrationValue // neutral for multiplication
	})
}

// normalizeBareCurrentValue replaces bare $ (not followed by a letter or underscore)
// with $value. Per MIOTY Application Layer Spec Table 15, bare $ means
// the current component value (e.g., "$/10" means "$value/10").
func (e *ExpressionEvaluator) normalizeBareCurrentValue(expr string) string {
	var result strings.Builder
	result.Grow(len(expr) + growSlack)
	for i := 0; i < len(expr); i++ {
		if expr[i] == '$' {
			next := i + 1
			if next >= len(expr) || (!isLetter(expr[next]) && expr[next] != '_') {
				result.WriteString(tokenValueRef)
				continue
			}
		}
		result.WriteByte(expr[i])
	}
	return result.String()
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// substituteSpecFieldRefs replaces spec-style $<name> references with their values.
// Per MIOTY Application Layer Spec Table 15, $<name> refers to another decoded field.
// This runs after $field.<name> and $calibration.<key> substitution to avoid conflicts.
func (e *ExpressionEvaluator) substituteSpecFieldRefs(expr string, fields map[string]interface{}) string {
	return e.specFieldRefPattern.ReplaceAllStringFunc(expr, func(match string) string {
		matches := e.specFieldRefPattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		fieldName := matches[1]
		// Skip keywords that are not field references
		if fieldName == keywordValue || fieldName == keywordField || fieldName == keywordCalibration {
			return match
		}
		if val, ok := fields[fieldName]; ok {
			if numVal, isNum := numconv.ToFloat64(val); isNum {
				return fmt.Sprintf("%v", numVal)
			}
			return fmt.Sprintf("%v", val)
		}
		return missingFieldValue // Default to 0 for missing fields
	})
}

// evaluateBooleanExpression evaluates a simple boolean expression.
// Supports: ==, !=, <, >, <=, >=, &&, ||, true, false
func (e *ExpressionEvaluator) evaluateBooleanExpression(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Handle literal boolean values
	if expr == literalTrue {
		return true, nil
	}
	if expr == literalFalse {
		return false, nil
	}

	// Handle && (AND) - split and evaluate both sides
	if strings.Contains(expr, "&&") {
		parts := strings.SplitN(expr, "&&", 2)
		left, err := e.evaluateBooleanExpression(parts[0])
		if err != nil {
			return false, err
		}
		right, err := e.evaluateBooleanExpression(parts[1])
		if err != nil {
			return false, err
		}
		return left && right, nil
	}

	// Handle || (OR) - split and evaluate both sides
	if strings.Contains(expr, "||") {
		parts := strings.SplitN(expr, "||", 2)
		left, err := e.evaluateBooleanExpression(parts[0])
		if err != nil {
			return false, err
		}
		right, err := e.evaluateBooleanExpression(parts[1])
		if err != nil {
			return false, err
		}
		return left || right, nil
	}

	// Handle comparison operators
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if strings.Contains(expr, op) {
			parts := strings.SplitN(expr, op, 2)
			if len(parts) == 2 {
				left := strings.TrimSpace(parts[0])
				right := strings.TrimSpace(parts[1])
				return e.compareValues(left, right, op)
			}
		}
	}

	// Try to parse as a number - non-zero is truthy
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num != 0, nil
	}

	return false, fmt.Errorf(errFmtCannotEvaluateExpr, expr)
}

// compareValues compares two string values with the given operator.
func (e *ExpressionEvaluator) compareValues(left, right, op string) (bool, error) {
	// Try numeric comparison first
	leftNum, leftErr := strconv.ParseFloat(left, 64)
	rightNum, rightErr := strconv.ParseFloat(right, 64)

	if leftErr == nil && rightErr == nil {
		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		}
	}

	// Fall back to string comparison for == and !=
	if op == "==" {
		return left == right, nil
	}
	if op == "!=" {
		return left != right, nil
	}

	return false, fmt.Errorf(errFmtCompareNonNumeric, op)
}

// evaluateArithmeticExpression evaluates a simple arithmetic expression.
// Supports: +, -, *, /, parentheses, and common math functions.
func (e *ExpressionEvaluator) evaluateArithmeticExpression(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, nil
	}

	// Simple recursive descent parser for arithmetic expressions
	result, _, err := e.parseExpression(expr, 0)
	return result, err
}

// parseExpression parses and evaluates an arithmetic expression.
func (e *ExpressionEvaluator) parseExpression(expr string, pos int) (float64, int, error) {
	// Skip whitespace
	for pos < len(expr) && expr[pos] == ' ' {
		pos++
	}

	// Parse term
	left, pos, err := e.parseTerm(expr, pos)
	if err != nil {
		return 0, pos, err
	}

	// Handle + and -
	for pos < len(expr) {
		for pos < len(expr) && expr[pos] == ' ' {
			pos++
		}
		if pos >= len(expr) {
			break
		}

		op := expr[pos]
		if op != '+' && op != '-' {
			break
		}
		pos++

		right, newPos, err := e.parseTerm(expr, pos)
		if err != nil {
			return 0, newPos, err
		}
		pos = newPos

		if op == '+' {
			left = left + right
		} else {
			left = left - right
		}
	}

	return left, pos, nil
}

// parseTerm parses a term (handles * and /).
func (e *ExpressionEvaluator) parseTerm(expr string, pos int) (float64, int, error) {
	left, pos, err := e.parseFactor(expr, pos)
	if err != nil {
		return 0, pos, err
	}

	for pos < len(expr) {
		for pos < len(expr) && expr[pos] == ' ' {
			pos++
		}
		if pos >= len(expr) {
			break
		}

		op := expr[pos]
		if op != '*' && op != '/' {
			break
		}
		pos++

		right, newPos, err := e.parseFactor(expr, pos)
		if err != nil {
			return 0, newPos, err
		}
		pos = newPos

		if op == '*' {
			left = left * right
		} else {
			if right == 0 {
				return 0, pos, errDivisionByZero
			}
			left = left / right
		}
	}

	return left, pos, nil
}

// parseFactor parses a factor (number, parenthesized expression, or function call).
func (e *ExpressionEvaluator) parseFactor(expr string, pos int) (float64, int, error) {
	for pos < len(expr) && expr[pos] == ' ' {
		pos++
	}

	if pos >= len(expr) {
		return 0, pos, errUnexpectedEndOfExpression
	}

	// Handle parentheses
	if expr[pos] == '(' {
		pos++
		result, pos, err := e.parseExpression(expr, pos)
		if err != nil {
			return 0, pos, err
		}
		for pos < len(expr) && expr[pos] == ' ' {
			pos++
		}
		if pos >= len(expr) || expr[pos] != ')' {
			return 0, pos, errMissingClosingParen
		}
		return result, pos + 1, nil
	}

	// Handle negative numbers
	negative := false
	if expr[pos] == '-' {
		negative = true
		pos++
		for pos < len(expr) && expr[pos] == ' ' {
			pos++
		}
	}

	// Handle math functions
	for _, fn := range mathFunctions {
		if strings.HasPrefix(expr[pos:], fn+"(") {
			pos += len(fn) + 1
			arg, newPos, err := e.parseExpression(expr, pos)
			if err != nil {
				return 0, newPos, err
			}
			pos = newPos
			for pos < len(expr) && expr[pos] == ' ' {
				pos++
			}
			if pos >= len(expr) || expr[pos] != ')' {
				return 0, pos, fmt.Errorf(errFmtMissingClosingParenFor, fn)
			}
			result := e.applyMathFunc(fn, arg)
			if negative {
				result = -result
			}
			return result, pos + 1, nil
		}
	}

	// Parse number
	start := pos
	for pos < len(expr) && (expr[pos] >= '0' && expr[pos] <= '9' || expr[pos] == '.') {
		pos++
	}

	if start == pos {
		return 0, pos, fmt.Errorf(errFmtExpectedNumberAtPos, pos)
	}

	num, err := strconv.ParseFloat(expr[start:pos], 64)
	if err != nil {
		return 0, pos, fmt.Errorf(errFmtInvalidNumber, expr[start:pos])
	}

	if negative {
		num = -num
	}
	return num, pos, nil
}

// applyMathFunc applies a math function to a value.
func (e *ExpressionEvaluator) applyMathFunc(fn string, val float64) float64 {
	switch fn {
	case fnSqrt:
		return math.Sqrt(val)
	case fnAbs:
		return math.Abs(val)
	case fnLog:
		return math.Log(val)
	case fnExp:
		return math.Exp(val)
	case fnSin:
		return math.Sin(val)
	case fnCos:
		return math.Cos(val)
	case fnTan:
		return math.Tan(val)
	default:
		return val
	}
}
