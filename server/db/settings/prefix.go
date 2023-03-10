package settings

import "time"

type prefixer struct {
	prefix string
}

func WithPrefix(prefix string) Settings {
	return &prefixer{prefix: prefix + "."}
}

func (p *prefixer) SetString(key string, val string) {
	SetString(p.prefix+key, val)
}
func (p *prefixer) SetInt(key string, val int64) {
	SetInt(p.prefix+key, val)
}
func (p *prefixer) SetFloat(key string, val float64) {
	SetFloat(p.prefix+key, val)
}
func (p *prefixer) SetTime(key string, val time.Time) {
	SetTime(p.prefix+key, val)
}
func (p *prefixer) SetBool(key string, val bool) {
	SetBool(p.prefix+key, val)
}
func (p *prefixer) SetJson(key string, val any) error {
	return SetJson(p.prefix+key, val)
}

func (p *prefixer) String(key string) (string, error) {
	return String(p.prefix + key)
}
func (p *prefixer) Int(key string) (int64, error) {
	return Int(p.prefix + key)
}
func (p *prefixer) Float(key string) (float64, error) {
	return Float(p.prefix + key)
}
func (p *prefixer) Time(key string) (time.Time, error) {
	return Time(p.prefix + key)
}
func (p *prefixer) Bool(key string) (bool, error) {
	return Bool(p.prefix + key)
}
func (p *prefixer) Json(key string, res any) error {
	return Json(p.prefix+key, &res)
}
