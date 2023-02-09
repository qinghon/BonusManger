module BonusManger

go 1.13

require (
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.7.7
	github.com/gorilla/websocket v1.4.1
	github.com/qinghon/network v0.0.0
	github.com/qinghon/system/bonus v0.0.0
	github.com/qinghon/system/tools v0.0.0
	github.com/sirupsen/logrus v1.5.0
	gopkg.in/yaml.v2 v2.2.8
)

replace (
	github.com/qinghon/hardware => ./src/hardware
	github.com/qinghon/network => ./src/network
	github.com/qinghon/system/bonus => ./src/system/bonus
	github.com/qinghon/system/package => ./src/system/package
	github.com/qinghon/system/tools => ./src/system/tools
	golang.org/x/net => github.com/golang/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sys => github.com/golang/sys v0.0.0-20200331124033-c3d80250170d
)
