package httperror

type Error struct {
	Message string `json:"error" example:"You must specify a transaction ID"`
}

func New(e error) Error {
	return Error{
		Message: e.Error(),
	}
}
