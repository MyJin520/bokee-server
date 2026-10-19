package verifyx

import (
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtrans "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
	"sync"
)

var (
	validate *validator.Validate
	uni      *ut.UniversalTranslator
	trans    ut.Translator
	once     sync.Once
	found    bool
)

func CheckStruct(obj any) string {
	once.Do(func() {
		validate = validator.New()
		translator := zh.New()
		uni = ut.New(translator, translator)
		trans, found = uni.GetTranslator(translator.Locale())
		if found { // 存在则注册翻译器
			_ = zhtrans.RegisterDefaultTranslations(validate, trans)
		}
	})
	validate.RegisterTagNameFunc(customTagNameFunc)
	err := validate.Struct(obj)
	return errList(err)

}

func errList(err error) string {
	if err != nil {
		errMsgList := make([]string, 0, 5)
		for _, fieldError := range err.(validator.ValidationErrors) {
			errMsgList = append(errMsgList, fieldError.Translate(trans))
		}
		errMsg := strings.Join(errMsgList, "; ")
		return errMsg
	}
	return ""
}

// 获取字段标签
func customTagNameFunc(field reflect.StructField) string {
	label := field.Tag.Get("label")
	if len(label) == 0 {
		return field.Name
	}
	return label
}
