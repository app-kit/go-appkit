package api


type ApiError struct {
	Code string
	Message string
}

func (a ApiError) Error() string {
	return a.Code + ": " + a.Message	
}
