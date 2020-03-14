module BonusManger

go 1.13

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.4.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/qinghon/hardware v0.0.0
	github.com/qinghon/network v0.0.0
	github.com/qinghon/system/bonus v0.0.0
	github.com/qinghon/system/tools v0.0.0
	github.com/sirupsen/logrus v1.4.2 // indirect
)

replace (
	github.com/qinghon/hardware => ./src/hardware
	github.com/qinghon/network => ./src/network
	github.com/qinghon/system/bonus => ./src/system/bonus
	github.com/qinghon/system/package => ./src/system/package
	github.com/qinghon/system/tools => ./src/system/tools
)
