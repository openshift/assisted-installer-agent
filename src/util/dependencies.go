package util

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/jaypipes/ghw"
	"github.com/vishvananda/netlink"
)

//go:generate mockery -name IDependencies -inpkg
type IDependencies interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
	ReadFile(fname string) ([]byte, error)
	Stat(fname string) (os.FileInfo, error)
	Hostname() (string, error)
	Interfaces() ([]Interface, error)
	Block(opts ...*ghw.WithOption) (*ghw.BlockInfo, error)
	Product(opts ...*ghw.WithOption) (*ghw.ProductInfo, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Abs(path string) (string, error)
	EvalSymlinks(path string) (string, error)
	LinkByName(name string) (netlink.Link, error)
	RouteList(link netlink.Link, family int) ([]netlink.Route, error)
}

type Dependencies struct {
	NetlinkRouteFinder
}

func (d *Dependencies) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return Execute(command, args...)
}

func (d *Dependencies) ReadFile(fname string) ([]byte, error) {
	return ioutil.ReadFile(fname)
}

func (d *Dependencies) Stat(fname string) (os.FileInfo, error) {
	return os.Stat(fname)
}

func (d *Dependencies) Hostname() (string, error) {
	return os.Hostname()
}

func (d *Dependencies) Interfaces() ([]Interface, error) {
	ins, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ret := make([]Interface, 0)
	for _, in := range ins {
		ret = append(ret, &NetworkInterface{netInterface: in, dependencies: d})
	}
	return ret, nil
}

func (d *Dependencies) Block(opts ...*ghw.WithOption) (*ghw.BlockInfo, error) {
	return ghw.Block(opts...)
}

func (d *Dependencies) Product(opts ...*ghw.WithOption) (*ghw.ProductInfo, error) {
	return ghw.Product(opts...)
}

func (d *Dependencies) ReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(dirname)
}

func (d *Dependencies) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (d *Dependencies) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func NewDependencies() IDependencies {
	return &Dependencies{}
}
