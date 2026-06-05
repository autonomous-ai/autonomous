package serializers

// JSONReponseStatus ...
type JSONReponseStatus int

const (
	// JSONReponseStatusFailed ...
	JSONReponseStatusFailed JSONReponseStatus = iota
	// JSONReponseStatusSuccess ...
	JSONReponseStatusSuccess
)

// JSONResponse ...
type JSONResponse struct {
	Status  JSONReponseStatus `json:"status"`
	Data    interface{}       `json:"data"`
	Message interface{}       `json:"message"`
}

// ResponseSuccess ...
func ResponseSuccess(data interface{}) JSONResponse {
	return JSONResponse{
		Status: JSONReponseStatusSuccess,
		Data:   data,
	}
}

// ResponseError ...
func ResponseError(err string) JSONResponse {
	return JSONResponse{
		Status:  JSONReponseStatusFailed,
		Message: err,
	}
}
