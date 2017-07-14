package server
import (
	"bufio"
	"net"
	"crabftp/log"
	"crabftp/config"
	"strings"
	"math/rand"
	"strconv"
	"errors"
)
var s *net.Listener
func GetIP() string {
	if config.ServerConfig.PublicIP==""{
		return strings.Split((*s).Addr().String(),":")[0]
	}
	return config.ServerConfig.PublicIP
}
func GetPASVConn() (net.Listener,error){
	i:=0
	for{
		i++
		port:=10000+rand.Intn(6000)
		l,err:=net.Listen("tcp4",":"+strconv.Itoa(port))
		if err!=nil{
			if i==100{
				return nil,errors.New("GetPASVPort Err")
			}
			continue
		}
		return l,nil
	}
}
func StartServer(){
	l,_:=net.Listen("tcp4",config.ServerConfig.PublicIP+":"+config.ServerConfig.ServerPort)
	s=&l
	log.NPrint("server","服务端启动成功，监听在"+config.ServerConfig.ServerPort+"端口")
	for{
		c,_:=l.Accept()
		log.DPrint("debug-server","客户端连接"+c.RemoteAddr().String())
		u:=User{
			Conn:c,
			Reader:bufio.NewReader(c),
			NowPath:"/",
			Permission:30,
		}
		u.WriteMsg("220","Hello!CrabFtp Server Deeeeesu!")
		go u.HandleConn()
	}
}
