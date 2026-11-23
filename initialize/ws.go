package initialize

import (
	"bokee/global"
	"bokee/internal/network/ws"
)

func initRealtime() {
	ws.Init(global.Config.IM)
}
