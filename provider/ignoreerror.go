package provider

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/evcc-io/evcc/util"
)

type ignoreerrorProvider struct {
	ignoreerror    func() (float64, error)
	log            *util.Logger
	config         Config
	errorstoignore []*regexp.Regexp
}

func init() {
	registry.Add("ignoreerror", NewIgnoreerrorFromConfig)
}

// NewConstFromConfig creates const provider
func NewIgnoreerrorFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		Ignoreerror    Config
		Errorstoignore []string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &ignoreerrorProvider{
		config: cc.Ignoreerror,
		log:    util.NewLogger("ignoreerror"),
	}

	for idx, errorstoignore := range cc.Errorstoignore {
		r, err := regexp.Compile(errorstoignore)
		if err != nil {
			return nil, fmt.Errorf("errorstoignore[%d]: %w", idx, err)
		}
		o.errorstoignore = append(o.errorstoignore, r)
	}

	err := tryCreate(o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func tryCreate(o *ignoreerrorProvider) error {
	f, err := NewFloatGetterFromConfig(o.config)
	if err != nil {
		ignoreError := false
		errortype := reflect.TypeOf(err).String()
		for idx, errorstoignore := range o.errorstoignore {
			if errorstoignore.MatchString(errortype) {
				ignoreError = true
				o.log.DEBUG.Println(fmt.Sprintf("error match[%d]", idx), errortype, err)
				break
			}
		}
		if ignoreError {
			o.log.WARN.Println("Ignore error:", errortype, err)
		} else {
			return fmt.Errorf("ignoreerror: %w", err)
		}
	} else {
		o.ignoreerror = f
	}
	return nil
}

func (o *ignoreerrorProvider) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		f, err := o.floatGetter()
		return int64(f), err
	}
}

func (o *ignoreerrorProvider) FloatGetter() func() (float64, error) {
	return o.floatGetter
}

func (o *ignoreerrorProvider) floatGetter() (float64, error) {
	if o.ignoreerror == nil {
		_ = tryCreate(o)
	}
	if o.ignoreerror != nil {
		return o.ignoreerror()
	}
	o.log.WARN.Println("Source not yet created, use 0")
	return 0, nil
}
