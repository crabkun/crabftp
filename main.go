package main

import (
	"crabftp/log"
	"crabftp/config"
	"crabftp/server"
)
func main(){
	log.StartLog(3)
	config.LoadConfig()
	server.StartServer()
}
