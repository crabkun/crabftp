package server

import (
	"net"
	"bufio"
	"fmt"
	"crabftp/util"
	"github.com/djimenez/iconv-go"
	"strings"
)

type User struct{
	Conn net.Conn
	Reader *bufio.Reader
	IsLogin bool
	UserName string
	Password string
	UserPath string
	NowPath string
	DataConn *net.Conn
	UTF8 bool
	RenameFrom string
	RenameTo string
	LastFilePos int64
	Permission int
}
func (u *User) CanRead() bool{
	return u.Permission%2==0
}
func (u *User) CanWrite() bool{
	return u.Permission%3==0
}
func (u *User) CanDelete() bool{
	return u.Permission%5==0
}
func (u *User) Disconnect(){
	u.Conn.Close()
}
func (u *User) Read()(string,string,error){
	//buf:=make([]byte,65536)
	//_,err:=u.Conn.Read(buf)
	buf,_,err:=u.Reader.ReadLine()
	if err!=nil{
		return "","",err
	}
	if !u.UTF8{
		out := make([]byte, len(buf)*2)
		_,l,_:=iconv.Convert(buf,out,"gb2312", "utf-8")
		buf=out[:l]
	}
	command,arg:="",""
	bufs:=util.B2s(buf)
	n,_:=fmt.Sscanf(bufs,"%s%s",&command,&arg)
	if n==0{
		return "","",err
	}
	if arg!=""{
		arg=strings.Replace(bufs,command+" ","",1)
	}
	return command,arg,nil
}
func (u *User) WriteMsg(code string,msg string)error{
	_,err:=u.Conn.Write([]byte(fmt.Sprintf("%s %s\r\n",code,msg)))
	return err
}
func (u *User) HandleConn() {
	for{
		command,args,err:=u.Read()
		if err!=nil{
			u.Disconnect()
			continue
		}
		runcommand(u,command,args)
	}
}