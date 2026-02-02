// Package query implements the query parsing and evaluation engine for Bundoc.
//
// Unstructured JSON queries (e.g., `{"age": {"$gt": 25}}`) are parsed into an
// Abstract Syntax Tree (AST), which is then used by the execution engine to filter
// documents.
package query

import (
	"fmt"
)

// Operator represents a comparison operator (e.g., $eq, $gt, $in).
type Operator string

const (
	OpEq  Operator = "$eq"
	OpNe  Operator = "$ne"
	OpGt  Operator = "$gt"
	OpGte Operator = "$gte"
	OpLt  Operator = "$lt"
	OpLte Operator = "$lte"
	OpIn  Operator = "$in"
)

// Node is the common interface for all nodes in the Query AST.
type Node interface {
	// Marker interface. Specific nodes (FieldNode, LogicalNode) implement logic.
}

// FieldNode represents a query on a specific field
type FieldNode struct {
	Field    string
	Operator Operator
	Value    interface{}
}

// LogicalNode represents AND/OR operations
type LogicalNode struct {
	Operator string // $and, $or
	Children []Node
}

// Parse converts a map-based query into an AST
// query: { "age": { "$gt": 25 }, "status": "active" }
func Parse(query map[string]interface{}) (Node, error) {
	var nodes []Node

	for key, val := range query {
		if key == "$and" || key == "$or" {
			// Handle logical operators
			list, ok := val.([]interface{})
			if !ok {
				return nil, fmt.Errorf("value for %s must be a list", key)
			}
			children := make([]Node, 0, len(list))
			for _, item := range list {
				subMap, ok := item.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("element of %s must be an object", key)
				}
				subNode, err := Parse(subMap)
				if err != nil {
					return nil, err
				}
				children = append(children, subNode)
			}
			nodes = append(nodes, &LogicalNode{Operator: key, Children: children})
		} else {
			// Handle field operators
			// check if val is a map like { "$gt": 25 }
			if valMap, ok := val.(map[string]interface{}); ok {
				for op, opVal := range valMap {
					// Validate operator
					switch Operator(op) {
					case OpEq, OpNe, OpGt, OpGte, OpLt, OpLte, OpIn:
						nodes = append(nodes, &FieldNode{Field: key, Operator: Operator(op), Value: opVal})
					default:
						return nil, fmt.Errorf("unknown operator: %s", op)
					}
				}
			} else {
				// Implicit $eq
				nodes = append(nodes, &FieldNode{Field: key, Operator: OpEq, Value: val})
			}
		}
	}

	return &LogicalNode{Operator: "$and", Children: nodes}, nil
}

// Matches checks if a document matches the node
func (n *FieldNode) Matches(doc map[string]interface{}) bool {
	val, exists := doc[n.Field]
	if !exists {
		return false
	}
	return compare(val, n.Operator, n.Value)
}

func (n *LogicalNode) Matches(doc map[string]interface{}) bool {
	if n.Operator == "$and" {
		for _, child := range n.Children {
			// Type assertion needed because Children is []Node
			if matcher, ok := child.(Matcher); ok {
				if !matcher.Matches(doc) {
					return false
				}
			}
		}
		return true
	}
	if n.Operator == "$or" {
		for _, child := range n.Children {
			if matcher, ok := child.(Matcher); ok {
				if matcher.Matches(doc) {
					return true
				}
			}
		}
		return false
	}
	return false
}

// Matcher interface
type Matcher interface {
	Matches(doc map[string]interface{}) bool
}

// Compare compares two values given an operator.
// Exposed for sorting logic.
// For sorting, we usually want -1, 0, 1.
// This function returns bool for AST matching.
// We need a separate CompareValues(a, b) int function.
func Compare(actual interface{}, op Operator, expected interface{}) bool {
	return compare(actual, op, expected)
}

// Helper comparison logic
func compare(actual interface{}, op Operator, expected interface{}) bool {
	// Simple type coercion/comparison logic needed here
	// For MVP, handle strings and numbers (float64 mostly due to JSON)

	switch op {
	case OpEq:
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case OpNe:
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case OpGt:
		// Need real number comparison
		return compareNumbers(actual, expected) > 0
	case OpLt:
		return compareNumbers(actual, expected) < 0
	}
	return false
}

// CompareValues returns -1 if a < b, 0 if a == b, 1 if a > b
func CompareValues(a, b interface{}) int {
	return compareNumbers(a, b)
}

func compareNumbers(a, b interface{}) int {
	// Attempt to cast to float64
	f1, ok1 := toFloat(a)
	f2, ok2 := toFloat(b)
	if ok1 && ok2 {
		if f1 > f2 {
			return 1
		}
		if f1 < f2 {
			return -1
		}
		return 0
	}
	// Fallback string compare
	s1 := fmt.Sprintf("%v", a)
	s2 := fmt.Sprintf("%v", b)
	if s1 > s2 {
		return 1
	}
	if s1 < s2 {
		return -1
	}
	return 0
}

func toFloat(v interface{}) (float64, bool) {
	switch i := v.(type) {
	case float64:
		return i, true
	case float32:
		return float64(i), true
	case int:
		return float64(i), true
	case int64:
		return float64(i), true
	}
	return 0, false
}
