package stdlib

import "reflect"

// go:generate yaegi extract fmt
// go:generate yaegi extract math
// go:generate yaegi extract strings
// go:generate yaegi extract time

// Symbols variable stores the map of stdlib symbols per package.
var Symbols = map[string]map[string]reflect.Value{}
