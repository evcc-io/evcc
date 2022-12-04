package assets

import "io/fs"

var (
	// Web is the embedded dist file system
	Web fs.FS

	// I18n is the embedded i18n file system
	I18n fs.FS
)

// Live indicates assets are passed-through from filesystem
func Live() bool {
	return Web != nil
}
