package inventory

import "os"

//go:generate mockery -name FileInfo -inpkg
type FileInfo interface {
	os.FileInfo
}
