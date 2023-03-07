package api

import (
	"simplebank/utils"

	"github.com/go-playground/validator/v10"
)

var validCurrency validator.Func = func(fl validator.FieldLevel) bool {
	// 先判断数据是否为字符串
	if currency, ok := fl.Field().Interface().(string); ok {
		return utils.IsSupportCurrency(currency)
	}
	return false
}
