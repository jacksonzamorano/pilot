package pilot

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusNoContent           StatusCode = 204
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusInternalServerError StatusCode = 500
)

var StatusCodeDescriptions = map[StatusCode]string{
	StatusOK:                  "OK",
	StatusNoContent:           "No Content",
	StatusBadRequest:          "Bad Request",
	StatusNotFound:            "Not Found",
	StatusUnauthorized:        "Unauthorized",
	StatusForbidden:           "Forbidden",
	StatusInternalServerError: "Internal Server Error",
}
