package main

import (
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"os"
)

var (
	log    = logging.MustGetLogger("example")
	format = logging.MustStringFormatter(`%{color}[%{level:.4s}] %{time:2006/01/02 - 15:04:05}%{color:reset} ▶ %{message}`)
)

func initLogger() {
	backend1 := logging.NewLogBackend(os.Stdout, "", 0)
	backend2 := logging.NewLogBackend(os.Stdout, "", 0)

	backend2Foramtter := logging.NewBackendFormatter(backend2, format)

	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")

	logging.SetBackend(backend1Leveled, backend2Foramtter)
}

func main() {
	//var port int
	//port = 80

	initLogger()
	log.Info("OpenAuth logger started")

	//	if os.Geteuid() != 0 {
	//		log.Error("No permission to run the program")
	//		return
	//	}

	err := os.Mkdir("/var/OpenAuth/accesslog", 0755)
	if err != nil && err == os.ErrExist {
		log.Error("Could not create access log directory")
	}

	gin.SetMode(gin.DebugMode)
	if err := StartServer("0.0.0.0:80"); err != nil {
		panic(err)
	}
}
