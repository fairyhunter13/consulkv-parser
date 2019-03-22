package consulparser

import "errors"

var (
	//ErrNilClient defines error for client that is nil.
	ErrNilClient = errors.New("client must not be nil")
	//ErrNonPointerType  defines error for the value that is non-pointer type.
	ErrNonPointerType = errors.New("value must be pointer type")
	//ErrUnhandledKind defines error for the kind that is not handled by this library.
	ErrUnhandledKind = errors.New("unhandled kind for assigning value to the field")
	//ErrOverflowSet defines error that will be used for overflow case.
	ErrOverflowSet = errors.New("error in set the overflowing value to the field")
)
