// Code generated for package testdata by go-bindata DO NOT EDIT. (@generated)
// sources:
// 202103221321_migration_1.up.sql
// 202103221400_migration_2.up.sql
// 202103221430_migration_3.up.sql
package testdata

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var __202103221321_migration_1UpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xca\xcd\x4c\x2f\x4a\x2c\xc9\xcc\xcf\x33\x04\x04\x00\x00\xff\xff\x9e\x50\x75\xf6\x0a\x00\x00\x00")

func _202103221321_migration_1UpSqlBytes() ([]byte, error) {
	return bindataRead(
		__202103221321_migration_1UpSql,
		"202103221321_migration_1.up.sql",
	)
}

func _202103221321_migration_1UpSql() (*asset, error) {
	bytes, err := _202103221321_migration_1UpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "202103221321_migration_1.up.sql", size: 10, mode: os.FileMode(420), modTime: time.Unix(1637940747, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __202103221400_migration_2UpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xca\xcd\x4c\x2f\x4a\x2c\xc9\xcc\xcf\x33\xe2\x02\x04\x00\x00\xff\xff\x43\x9e\xbb\x0e\x0b\x00\x00\x00")

func _202103221400_migration_2UpSqlBytes() ([]byte, error) {
	return bindataRead(
		__202103221400_migration_2UpSql,
		"202103221400_migration_2.up.sql",
	)
}

func _202103221400_migration_2UpSql() (*asset, error) {
	bytes, err := _202103221400_migration_2UpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "202103221400_migration_2.up.sql", size: 11, mode: os.FileMode(420), modTime: time.Unix(1637940747, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __202103221430_migration_3UpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xca\xcd\x4c\x2f\x4a\x2c\xc9\xcc\xcf\x33\xe6\x02\x04\x00\x00\xff\xff\x02\xaf\xa0\x17\x0b\x00\x00\x00")

func _202103221430_migration_3UpSqlBytes() ([]byte, error) {
	return bindataRead(
		__202103221430_migration_3UpSql,
		"202103221430_migration_3.up.sql",
	)
}

func _202103221430_migration_3UpSql() (*asset, error) {
	bytes, err := _202103221430_migration_3UpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "202103221430_migration_3.up.sql", size: 11, mode: os.FileMode(420), modTime: time.Unix(1637940747, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"202103221321_migration_1.up.sql": _202103221321_migration_1UpSql,
	"202103221400_migration_2.up.sql": _202103221400_migration_2UpSql,
	"202103221430_migration_3.up.sql": _202103221430_migration_3UpSql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//
//	data/
//	  foo.txt
//	  img/
//	    a.png
//	    b.png
//
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"202103221321_migration_1.up.sql": {_202103221321_migration_1UpSql, map[string]*bintree{}},
	"202103221400_migration_2.up.sql": {_202103221400_migration_2UpSql, map[string]*bintree{}},
	"202103221430_migration_3.up.sql": {_202103221430_migration_3UpSql, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
