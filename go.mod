module BonusManger

go 1.13

require (
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.4.0
	github.com/gorilla/websocket v1.4.1
	github.com/qinghon/network v0.0.0
	github.com/qinghon/system/bonus v0.0.0
	github.com/qinghon/system/tools v0.0.0
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a // indirect
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527 // indirect
)

replace (
	github.com/qinghon/hardware => ./src/hardware
	github.com/qinghon/network => ./src/network
	github.com/qinghon/system/bonus => ./src/system/bonus
	github.com/qinghon/system/package => ./src/system/package
	github.com/qinghon/system/tools => ./src/system/tools
)
