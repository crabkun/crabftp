package config

import (
	"io/ioutil"
	"crabftp/log"
	"os"
	"encoding/json"
)

type ConfigStruct struct{
	ServerPort string
	PublicIP string
	FtpPath string
}
var ServerConfig ConfigStruct
func LoadConfig(){
	buf,err:=ioutil.ReadFile("config.json")
	if err!=nil{
		log.NPrint("config","配置文件config.json读取错误，请参照config_example.json来修改")
		os.Exit(-1)
	}

	err=json.Unmarshal(buf,&ServerConfig)
	if err!=nil{
		log.NPrint("config","配置文件config.json格式错误，请参照config_example.json来修改")
		os.Exit(-1)
	}
	log.NPrint("config","配置文件读取成功")
	_,err=os.Stat(ServerConfig.FtpPath)
	if err!=nil{
		log.NPrint("config","配置中的FTP目录错误或者不存在，请手动创建")
		os.Exit(-1)
	}
}