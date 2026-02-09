package bundoc

import "errors"

var (
	ErrReferenceTargetNotFound   = errors.New("reference target not found")
	ErrReferenceRestrictViolation = errors.New("reference restrict violation")
	ErrInvalidReferenceSchema     = errors.New("invalid reference schema")
	ErrSchemaOverrideBlocked      = errors.New("schema override is disabled for this collection")
)
