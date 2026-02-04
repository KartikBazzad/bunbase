package rules

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

// AuthContext represents the authentication state of the request
type AuthContext struct {
	UID     string                 `json:"uid"`
	Claims  map[string]interface{} `json:"claims"`
	IsAdmin bool                   `json:"-"` // Internal flag, not exposed to CEL directly? Or exposed as request.auth.isAdmin?
}

// RuleContext represents the context available to a rule
type RuleContext struct {
	Auth     *AuthContext           `json:"auth"`
	Resource map[string]interface{} `json:"resource"` // The document
	Request  map[string]interface{} `json:"request"`  // Incoming data/params
}

// RulesEngine handles compilation and evaluation of CEL rules
type RulesEngine struct {
	env      *cel.Env
	prgCache sync.Map // map[string]cel.Program
}

// NewRulesEngine creates a new RulesEngine with standard environment
func NewRulesEngine() (*RulesEngine, error) {
	// Define the environment options
	// Variables:
	// - request: { auth: { uid: string, claims: map }, time: timestamp, resource: { data: map } }
	// - resource: { data: map, id: string }

	// Simplifying for MVP:
	// request.auth.uid
	// resource.data

	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("request", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("resource", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		return nil, err
	}

	return &RulesEngine{
		env: env,
	}, nil
}

// Evaluate evaluates a rule expression against a context
func (re *RulesEngine) Evaluate(expression string, ctx map[string]interface{}) (bool, error) {
	if expression == "" {
		return false, nil // Default deny? Or allow? Firestore defaults deny.
	}
	if expression == "true" {
		return true, nil
	}
	if expression == "false" {
		return false, nil
	}

	// Check cache
	var prg cel.Program
	if val, ok := re.prgCache.Load(expression); ok {
		prg = val.(cel.Program)
	} else {
		// Compile
		ast, issues := re.env.Compile(expression)
		if issues != nil && issues.Err() != nil {
			return false, fmt.Errorf("compile error: %s", issues.Err())
		}

		p, err := re.env.Program(ast)
		if err != nil {
			return false, fmt.Errorf("program construction error: %s", err)
		}
		prg = p
		re.prgCache.Store(expression, prg)
	}

	// Evaluate
	out, _, err := prg.Eval(ctx)
	if err != nil {
		return false, fmt.Errorf("eval error: %s", err)
	}

	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("rule must return boolean")
	}

	return result, nil
}
