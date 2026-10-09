package middleware

import (
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"KeuskupanLaboanBajo_BE/dto"
)

var validate = validator.New()

func ErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	resp := dto.ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: "Terjadi kesalahan internal",
	}

	switch e := err.(type) {
	case *echo.HTTPError:
		code = e.Code
		resp.Code = "HTTP_ERROR"
		if s, ok := e.Message.(string); ok {
			resp.Message = s
		} else {
			resp.Message = "HTTP error"
		}
		if e.Internal != nil {
			log.Printf("[ERROR] internal error: %v", e.Internal)
			if code >= http.StatusInternalServerError {
				resp.Message = "Terjadi kesalahan internal"
			}
		}
	case validator.ValidationErrors:
		code = http.StatusBadRequest
		resp.Code = "VALIDATION_ERROR"
		resp.Message = "Validasi gagal"
		resp.Details = make([]dto.ErrorDetail, len(e))
		for i, fe := range e {
			resp.Details[i] = dto.ErrorDetail{
				Field:   fe.Field(),
				Message: validationMsg(fe),
			}
		}
	default:
		log.Printf("[ERROR] unhandled error: %v", err)
		resp.Message = "Terjadi kesalahan internal"
	}

	if !c.Response().Committed {
		c.JSON(code, resp)
	}
}

func validationMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "Field ini wajib diisi"
	case "email":
		return "Format email tidak valid"
	case "min":
		return "Minimal " + fe.Param() + " karakter"
	case "max":
		return "Maksimal " + fe.Param() + " karakter"
	case "dive":
		return "Data tidak valid"
	default:
		return "Tidak valid"
	}
}

type CustomValidator struct {
	Validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.Validator.Struct(i)
}
