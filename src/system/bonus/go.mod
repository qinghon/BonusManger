module github.com/qinghon/system/bonus

go 1.12

require (
	github.com/qinghon/network v0.0.0
	github.com/qinghon/system/tools v0.0.0
	github.com/sirupsen/logrus v1.5.0

)

replace (
	github.com/qinghon/network => ../../network
	github.com/qinghon/system/tools => ../tools
)
