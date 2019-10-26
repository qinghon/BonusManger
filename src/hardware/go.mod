module github.com/qinghon/hardware

go 1.12

require (
	github.com/jaypipes/ghw v0.0.0-20190821154021-743802778342
	github.com/pkg/errors v0.8.0
	github.com/qinghon/system/package v0.0.0
	github.com/shirou/gopsutil v2.19.9+incompatible
	golang.org/x/sys v0.0.0-20191022100944-742c48ecaeb7 // indirect
)

replace github.com/qinghon/system/package => ../system/package
