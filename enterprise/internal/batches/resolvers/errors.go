package resolvers

import "fmt"

type ErrInvalidFirstParameter struct {
	Min, Max, First int
}

func (e ErrInvalidFirstParameter) Error() string {
	return fmt.Sprintf("first param %d is out of range (min=%d, max=%d)", e.First, e.Min, e.Max)
}

func (e ErrInvalidFirstParameter) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrInvalidFirstParameter"}
}

type ErrIDIsZero struct{}

func (e ErrIDIsZero) Error() string {
	return "invalid node id"
}

func (e ErrIDIsZero) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrIDIsZero"}
}

type ErrBatchChangesDisabled struct{ error }

func (e ErrBatchChangesDisabled) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrBatchChangesDisabled"}
}

type ErrBatchChangesDisabledForUser struct{ error }

func (e ErrBatchChangesDisabledForUser) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrBatchChangesDisabledForUser"}
}

// ErrBatchChangesUnlicensed wraps an underlying licensing.featureNotActivatedError
// to add an error code.
type ErrBatchChangesUnlicensed struct{ error }

func (e ErrBatchChangesUnlicensed) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrBatchChangesUnlicensed"}
}

type ErrBatchChangesDisabledDotcom struct{ error }

func (e ErrBatchChangesDisabledDotcom) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrBatchChangesDisabledDotcom"}
}

type ErrEnsureBatchChangeFailed struct{}

func (e ErrEnsureBatchChangeFailed) Error() string {
	return "a batch change in the given namespace and with the given name exists but does not match the given ID"
}

func (e ErrEnsureBatchChangeFailed) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrEnsureBatchChangeFailed"}
}

type ErrApplyClosedBatchChange struct{}

func (e ErrApplyClosedBatchChange) Error() string {
	return "existing batch change matched by batch spec is closed"
}

func (e ErrApplyClosedBatchChange) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrApplyClosedBatchChange"}
}

type ErrMatchingBatchChangeExists struct{}

func (e ErrMatchingBatchChangeExists) Error() string {
	return "a batch change matching the given batch spec already exists"
}

func (e ErrMatchingBatchChangeExists) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrMatchingBatchChangeExists"}
}

type ErrDuplicateCredential struct{}

func (e ErrDuplicateCredential) Error() string {
	return "a credential for this code host already exists"
}

func (e ErrDuplicateCredential) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrDuplicateCredential"}
}

type ErrVerifyCredentialFailed struct {
	SourceErr error
}

func (e ErrVerifyCredentialFailed) Error() string {
	return fmt.Sprintf("Failed to verify the credential:\n```\n%s\n```\n", e.SourceErr)
}

func (e ErrVerifyCredentialFailed) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "ErrVerifyCredentialFailed"}
}
