package error

type Error interface {
	GetCode() string
	GetMessage() string
	GetData() interface{}

	IsInternal() bool

	GetErrors() []error
	AddError(error)

	Error() string
}
