package services

import "errors"

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrEmailInUse = errors.New("email already in use")

	// ErrFolderHasDocs and ErrFolderHasChildren are returned by DeleteEmptyFolder
	// when the folder still contains documents or sub-folders, respectively.
	ErrFolderHasDocs     = errors.New("folder has documents")
	ErrFolderHasChildren = errors.New("folder has child folders")

	// ErrShiftCancelled is returned when a management operation targets a cancelled shift.
	ErrShiftCancelled = errors.New("shift is cancelled")
)

// ValidationError carries a human-readable message from input validation.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// validationErr constructs a ValidationError. Used across all services.
func validationErr(msg string) error { return &ValidationError{Message: msg} }

// AsValidationError unwraps err as *ValidationError; returns nil if it is not one.
func AsValidationError(err error) *ValidationError {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve
	}
	return nil
}
