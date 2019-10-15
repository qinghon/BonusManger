module github.com/qinghon/hardware

go 1.12

require (
	github.com/jaypipes/ghw v0.0.0-20190821154021-743802778342
	github.com/qinghon/system/package v0.0.0
	github.com/shirou/gopsutil v2.19.9+incompatible
)

replace github.com/qinghon/system/package => ../system/package
