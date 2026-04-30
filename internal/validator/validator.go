package validator

import (
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	playgroundvalidator "github.com/go-playground/validator/v10"
)

var (
	registerOnce    sync.Once
	inviteCodeRegex = regexp.MustCompile(`^[A-Z0-9]{6,20}$`)
)

// Register 将自定义校验规则注册到 Gin 默认绑定器。
func Register() error {
	var registerErr error
	registerOnce.Do(func() {
		validate, ok := binding.Validator.Engine().(*playgroundvalidator.Validate)
		if !ok {
			return
		}

		validate.RegisterTagNameFunc(func(field reflect.StructField) string {
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				return field.Name
			}
			name := strings.Split(jsonTag, ",")[0]
			if name == "" {
				return field.Name
			}
			return name
		})

		if err := validate.RegisterValidation("note_type", func(fl playgroundvalidator.FieldLevel) bool {
			switch fl.Field().String() {
			case "normal", "burn", "timed":
				return true
			default:
				return false
			}
		}); err != nil {
			registerErr = err
			return
		}

		if err := validate.RegisterValidation("invite_code", func(fl playgroundvalidator.FieldLevel) bool {
			return inviteCodeRegex.MatchString(fl.Field().String())
		}); err != nil {
			registerErr = err
			return
		}
	})

	return registerErr
}
