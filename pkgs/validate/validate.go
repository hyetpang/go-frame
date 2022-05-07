package validate

import "github.com/go-playground/validator/v10"

func MustValidate(a any) {
	if err := validator.New().Struct(a); err != nil {
		panic(err)
	}
}
