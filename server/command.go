package server

import (
	"path/filepath"
	"strings"
	"fmt"
	"crabftp/log"
	"strconv"
	"net"
	"io/ioutil"
	"crabftp/config"
	"os"
	"github.com/djimenez/iconv-go"
	"errors"
	"io"
)

func BuildPath(filename string) (fullPath string) {
	if len(filename) > 0 && filename[0:1] == "/" {
		fullPath = filepath.Clean(filename)
	} else {
		fullPath = filepath.Clean("/"+filename)
	}
	fullPath = strings.Replace(fullPath, "//", "/", -1)
	fullPath = strings.Replace(fullPath, string(filepath.Separator), "/", -1)
	return
}
func USER(c *User,args string){
	c.UserName=args
	c.WriteMsg("331","User name ok, password required")
}

func PASS(c *User,args string){
	c.Password=args
	if auth(c){
		c.WriteMsg("230","Password ok, continue")
		c.IsLogin=true
		log.DPrint("ftp",fmt.Sprintf("客户端%s成功登录%s",c.Conn.RemoteAddr(),c.UserName))
	}else{
		c.WriteMsg("530","Incorrect password, not logged in")
		c.Disconnect()
	}
}
func PASVAcc(c *User,l *net.Listener){
	conn,err:=(*l).Accept()
	if err!=nil{
		return
	}
	c.DataConn=&conn
}
func PASV(c *User){
	IP:=GetIP()
	IParr:=strings.Split(IP,".")
	if len(IParr)!=4{
		log.NPrint("PASV","被动连接端口建立错误，原因是IP格式错误")
		return
	}
	pasv,err:=GetPASVConn()
	if err!=nil{
		log.NPrint("PASV","无法给客户端分配被动连接端口")
		pasv.Close()
		return
	}
	portstr:=strings.Split(pasv.Addr().String(),":")[1]
	port,_:=strconv.Atoi(portstr)
	log.DPrint("PASV",fmt.Sprintf("成功给客户端%s分配被动连接端口%s",c.Conn.LocalAddr().String(),IP+":"+portstr))
	c.WriteMsg("227",fmt.Sprintf("Entering Passive Mode (%s,%s,%s,%s,%d,%d)",IParr[0],IParr[1],IParr[2],IParr[3],port/256,port%256))
	go PASVAcc(c,&pasv)
}
func lpad(input string, length int) (result string) {
	if len(input) < length {
		result = strings.Repeat(" ", length-len(input)) + input
	} else if len(input) == length {
		result = input
	} else {
		result = input[0:length]
	}
	return
}
func LIST(c *User){
	p:=BuildPath(c.NowPath)
	c.WriteMsg("150", "Opening ASCII mode data connection for file list")
	files,err:=ioutil.ReadDir(config.ServerConfig.FtpPath+p)
	i:=0
	buf:=""
	defer func(){
		msg := "Closing data connection, sent " + strconv.Itoa(i) + " bytes"
		c.WriteMsg("226",msg)
	}()
	if err!=nil{
		return
	}
	buf+=fmt.Sprintf("drwxrwxrwx 1 root root 1 Jan 12 15:04 .\r\n")
	buf+=fmt.Sprintf("drwxrwxrwx 1 root root 1 Jan 12 15:04 ..\r\n")
	for _,file:=range files{
		buf+=fmt.Sprintf("%s 1 root root %s %s %s\r\n",
			file.Mode().String(),
			lpad(strconv.Itoa(int(file.Size())), 12),
			file.ModTime().Format("Jan _2 15:04"),
			file.Name(),
		)
	}
	i=len(buf)
	SendToDataConn(c,[]byte(buf))
}
func SendToDataConn(c *User,data []byte){
	defer func(){
		if c.DataConn!=nil{
			(*c.DataConn).Close()
			c.DataConn=nil
		}
	}()
	if c.DataConn!=nil{
		if !c.UTF8{
			op:=make([]byte,len(data))
			_,n,_:=iconv.Convert(data,op,"utf-8","gb2312")
			data=op[:n]
		}
		(*c.DataConn).Write(data)
	}
}
func CWD(c *User,arg string){
	path:=BuildPath(c.NowPath)
	if arg[:1]!="/"{
		path+="/"+arg
	}else{
		path=arg
	}
	path=BuildPath(path)
	dir,err:=os.Stat(config.ServerConfig.FtpPath+path)
	if err != nil || !dir.IsDir(){
		c.WriteMsg("450","Action not taken")
	} else {
		c.WriteMsg("250", "Directory changed to "+path)
		c.NowPath=path
	}
}
func RNFR(c *User,arg string){
	if !c.CanWrite() {
		c.WriteMsg("450", "Permission denied")
		return
	}
	if arg==""{
		c.WriteMsg("450","Action not taken")
		return
	}
	path:=BuildPath(arg)
	pos:=strings.LastIndex(path,"/")+1
	c.RenameFrom=path[pos:]
	c.WriteMsg("350","Requested file action pending further information.")
}
func RNTO(c *User,arg string){
	if !c.CanWrite() {
		c.WriteMsg("450", "Permission denied")
		return
	}
	if arg==""||arg==c.RenameFrom{
		c.WriteMsg("450","Action not taken")
		return
	}
	path:=BuildPath(arg)
	c.RenameTo=path
	err:=os.Rename(config.ServerConfig.FtpPath+c.NowPath+"/"+c.RenameFrom,config.ServerConfig.FtpPath+c.NowPath+"/"+c.RenameTo)
	if err!=nil{
		c.WriteMsg("450","Action not taken")
	}else{
		c.WriteMsg("250","File renamed")
	}
	log.DPrint("ftp",fmt.Sprintf("用户%v（IP:%v）把%s改名或者移动成%s",c.UserName,c.Conn.RemoteAddr(),c.RenameFrom,c.RenameTo))
}
func DEL(c *User,arg string){
	if !c.CanDelete(){
		c.WriteMsg("450", "Permission denied")
		return
	}
	if arg==""{
		c.WriteMsg("450","Action not taken")
		return
	}
	path:=BuildPath(arg)
	pos:=strings.LastIndex(path,"/")+1
	delfile:=path[pos:]
	err:=os.RemoveAll(config.ServerConfig.FtpPath+c.NowPath+"/"+delfile)
	if err!=nil{
		c.WriteMsg("450","Action not taken "+err.Error())
	}else{
		c.WriteMsg("250","File deleted")
	}
	log.DPrint("ftp",fmt.Sprintf("用户%v（IP:%v）删除了%s",c.UserName,c.Conn.RemoteAddr(),arg))
}
func MKD(c *User,arg string){
	if !c.CanWrite() {
		c.WriteMsg("450", "Permission denied")
		return
	}
	path:=BuildPath(arg)
	pos:=strings.LastIndex(path,"/")+1
	newdir:=path[pos:]
	err:=os.Mkdir(config.ServerConfig.FtpPath+c.NowPath+"/"+newdir,0666)
	if err == nil {
		c.WriteMsg("257", "Directory created")
	} else {
		c.WriteMsg("550", "Action not taken")
	}
	log.DPrint("ftp",fmt.Sprintf("用户%v（IP:%v）新建了目录%s",c.UserName,c.Conn.RemoteAddr(),arg))
}
func checkFile(c *User,args string) error {
	args=BuildPath(args)
	path:=config.ServerConfig.FtpPath+c.NowPath+args
	f,err:=os.Stat(path)
	if err!=nil{
		return nil
	}
	if f.IsDir(){
		return errors.New("it's a dir")
	}
	return nil
}
func StartFileRecv(c *User,args string) (int,error){
	args=BuildPath(args)
	path:=config.ServerConfig.FtpPath+c.NowPath+args
	if c.DataConn==nil{
		return 0,errors.New("pasv failed")
	}
	defer func(){
		(*c.DataConn).Close()
		c.DataConn=nil
	}()
	buf:=make([]byte,65536)
	f,err:=os.OpenFile(path,os.O_CREATE|os.O_TRUNC,0666)
	if err!=nil{
		return 0,errors.New("file open failed")
	}
	i:=0
	for{
		n,err:=(*c.DataConn).Read(buf)
		i+=n
		if err!=nil{
			return i,nil
		}
		f.Write(buf[:n])
	}
	f.Sync()
	f.Close()
	return i,nil
}
func STOR(c *User,args string){
	if !c.CanWrite() {
		c.WriteMsg("450", "Permission denied")
		return
	}
	if err:=checkFile(c,args);err!=nil{
		c.WriteMsg("450", "Action not taken because "+err.Error())
		return
	}
	c.WriteMsg("150", "Data transfer starting")
	n,err:=StartFileRecv(c,args)
	if err == nil {
		msg := "OK, received " + strconv.Itoa(int(n)) + " bytes"
		c.WriteMsg("226", msg)
		log.DPrint("ftp",fmt.Sprintf("用户%v（IP:%v）上传了%s",c.UserName,c.Conn.RemoteAddr(),args))
	} else {
		c.WriteMsg("450", "Action not taken")
	}

}
func REST(c *User,args string){
	n,err:=strconv.ParseInt(args,10,0)
	if err!=nil{
		c.WriteMsg("500","REST Failed")
		return
	}
	c.LastFilePos=n
	c.WriteMsg("350", fmt.Sprint("Start transfer from ", c.LastFilePos))
}
func RETR(c *User,args string){
	defer func(){
		c.LastFilePos=0
	}()
	if !c.CanRead(){
		c.WriteMsg("450", "Permission denied")
		return
	}
	args=BuildPath(args)
	path:=config.ServerConfig.FtpPath+c.NowPath+args
	f,err:=os.OpenFile(path,os.O_RDONLY,0666)
	if err!=nil || c.DataConn==nil{
		c.WriteMsg("450", "File not available")
		return
	}
	defer func(){
		(*c.DataConn).Close()
		c.DataConn=nil
		f.Close()
	}()
	finfo,_:=f.Stat()
	if c.LastFilePos!=0{
		_,err:=f.Seek(c.LastFilePos,0)
		if err!=nil{
			return
		}
	}
	c.WriteMsg("150",fmt.Sprintf("Data transfer starting %v bytes",finfo.Size() ))
	b,_:=io.Copy(*c.DataConn,f)
	if err == nil {
		log.DPrint("ftp",fmt.Sprintf("用户%v（IP:%v）下载了%s",c.UserName,c.Conn.RemoteAddr(),args))
		message := fmt.Sprintf("Closing data connection, sent %v bytes",b)
		c.WriteMsg("226", message)
	} else {
		c.WriteMsg("450", "Action not taken")
	}
}
func SIZE(c *User,args string){
	args=BuildPath(args)
	if args=="/"{
		c.WriteMsg("213", "0")
		return
	}
	path:=config.ServerConfig.FtpPath+c.NowPath+args
	stat, err := os.Stat(path)
	if err != nil {
		c.WriteMsg("450", err.Error())
	} else {
		c.WriteMsg("213", strconv.Itoa(int(stat.Size())))
	}
}
func runcommand(u *User,command string,args string){
	command=strings.ToUpper(command)
	switch command {
	case "USER":
		USER(u,args)
	case "PASS":
		PASS(u,args)
	case "NOOP":
		u.WriteMsg("200","OK")
	default:
		if !u.IsLogin{
			u.WriteMsg("332","Need Login")
			return
		}
		switch command {
		case "TYPE":
			if strings.ToUpper(args) == "A" {
				u.WriteMsg("200", "Type set to ASCII")
			} else if strings.ToUpper(args) == "I" {
				u.WriteMsg("200", "Type set to binary")
			} else {
				u.WriteMsg("500", "Invalid type")
			}
		case "SYST":
			u.WriteMsg("215","UNIX Type: L8")
		case "FEAT":
			u.WriteMsg("221","Extensions supported:\n211 END")
		case "PWD":
			u.NowPath=BuildPath(u.NowPath)
			u.WriteMsg("257", fmt.Sprintf("\"%s\" is the current directory",u.NowPath))
		case "PASV":
			PASV(u)
		case "PORT":
			u.WriteMsg("500", "不支持主动模式！请切换到被动(PASV)模式")
		case "LIST":
			LIST(u)
		case "CWD":
			CWD(u,args)
		case "RNFR"	:
			RNFR(u,args)
		case "RNTO"	:
			RNTO(u,args)
		case "DELE":
			DEL(u,args)
		case "CDUP":
			CWD(u,"..")
		case "MKD":
			MKD(u,args)
		case "STOR":
			STOR(u,args)
		case "RETR":
			RETR(u,args)
		case "REST":
			REST(u,args)
		case "SIZE":
			SIZE(u,args)
		case "QUIT":
			u.WriteMsg("221", "Goodbye")
			u.Disconnect()
		case "RMD":
			DEL(u,args)
		default:
			u.WriteMsg("500","unsupported command")
		}
	}
}