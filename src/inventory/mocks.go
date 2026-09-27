package inventory

import "os"

//go:generate mockery --name FileInfo --inpackage
type FileInfo interface {
	os.FileInfo
}
