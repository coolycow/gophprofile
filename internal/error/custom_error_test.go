package error_test

import (
	"testing"

	profileError "github.com/coolycow/gophprofile/internal/error"

	"github.com/stretchr/testify/assert"
)

// Проверяем, что CustomError реализует error и отдаёт сообщение клиенту.
func TestCustomError_Error(t *testing.T) {
	e := profileError.CustomError{Message: "oops", StatusCode: 404}
	assert.Equal(t, "oops", e.Error())
}
