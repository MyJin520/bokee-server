package initialize

import (
	"bokee/config"
	"bokee/internal/network/router"
	"bokee/pkg/casbinx"
)

func ginServerInit(systemConfig config.SystemConfig) {
	engine := router.Routers()
	casbinx.InitSuperRoleCasbin(engine)
	casbinx.InitDefaultUserRoleCasbin(engine)
	err := engine.Run(systemConfig.Addr)
	if err != nil {
		panic("服务启动异常> " + err.Error())
	}
}
