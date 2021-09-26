package configs

import "strings"

// Version is replaced in compile time
// `-ldflags "-X 'github.com/LucienZhang/goto/configs.Version=${VERSION}'"`
var Version = "0.0.1"

// GetVersion returns escaped version number
func GetVersion() string {
	return strings.Replace(Version, " ", "-", -1)
}
