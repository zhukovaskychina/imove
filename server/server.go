package server

import (
	"github.com/gin-gonic/gin"
	"imove-server/server/conf"
)

type IMoveServer struct {
	Engine          *gin.Engine
	Cfg             *conf.Cfg
}

func  NewIMoveServer() *IMoveServer {
	engine := gin.Default()
	return &IMoveServer{

		Engine:         engine,
		Cfg:            conf.NewCfg(),
	}
}
