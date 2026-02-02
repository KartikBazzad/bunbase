package query

import (
	"testing"
)

func TestParseAndMatch(t *testing.T) {
	// 1. Test Simple Equity
	// { "role": "admin" }
	q1 := map[string]interface{}{
		"role": "admin",
	}
	ast1, err := Parse(q1)
	if err != nil {
		t.Fatalf("Failed to parse q1: %v", err)
	}

	doc1 := map[string]interface{}{"role": "admin", "age": 30}
	doc2 := map[string]interface{}{"role": "user", "age": 25}

	matcher1 := ast1.(Matcher)
	if !matcher1.Matches(doc1) {
		t.Errorf("Doc1 should match q1")
	}
	if matcher1.Matches(doc2) {
		t.Errorf("Doc2 should not match q1")
	}

	// 2. Test Comparison
	// { "age": { "$gt": 25 } }
	q2 := map[string]interface{}{
		"age": map[string]interface{}{"$gt": 25},
	}
	ast2, err := Parse(q2)
	if err != nil {
		t.Fatal(err)
	}
	matcher2 := ast2.(Matcher)
	if !matcher2.Matches(doc1) { // age 30
		t.Errorf("Doc1 (30) > 25")
	}
	if matcher2.Matches(doc2) { // age 25
		t.Errorf("Doc2 (25) is not > 25")
	}

	// 3. Test Logical AND
	// { "role": "admin", "age": { "$gt": 20 } }
	q3 := map[string]interface{}{
		"role": "admin",
		"age":  map[string]interface{}{"$gt": 20},
	}
	ast3, err := Parse(q3)
	if err != nil {
		t.Fatal(err)
	}
	matcher3 := ast3.(Matcher)
	if !matcher3.Matches(doc1) {
		t.Errorf("Doc1 should match q3")
	}
	if matcher3.Matches(doc2) {
		t.Errorf("Doc2 mismatch role")
	}
}
