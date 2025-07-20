package javascript

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
	"github.com/samber/lo"
)

var (
	mu       sync.Mutex
	registry = make(map[string]*otto.Otto)
)

// expose mutex to serialize VM access
func Lock() {
	mu.Lock()
}

// expose mutex to serialize VM access
func Unlock() {
	mu.Unlock()
}

// RegisteredVM returns a JS VM. If name is not empty, it will return a shared instance.
func RegisteredVM(name, init string) (*otto.Otto, error) {
	mu.Lock()
	defer mu.Unlock()

	name = strings.ToLower(name)
	vm, ok := registry[name]

	// create new VM
	if !ok {
		vm = otto.New()
		if err := setConsole(vm, name); err != nil {
			return nil, err
		}

		if init != "" {
			if _, err := vm.Run(init); err != nil {
				return nil, err
			}
		}

		if name != "" {
			registry[name] = vm
		}
	}

	return vm, nil
}

func setConsole(vm *otto.Otto, suffix string) error {
	name := "js"
	if suffix != "" {
		name = name + "-" + suffix
	}

	log := util.NewLogger(name)

	console := map[string]any{
		"trace": printer(log.TRACE),
		"log":   printer(log.DEBUG),
		"info":  printer(log.INFO),
		"error": printer(log.ERROR),
	}

	return vm.Set("console", console)
}

func printer(log *log.Logger) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		output := lo.Map(call.ArgumentList, func(a otto.Value, _ int) string {
			return fmt.Sprintf("%v", a)
		})

		log.Println(strings.Join(output, " "))

		return otto.UndefinedValue()
	}
}
