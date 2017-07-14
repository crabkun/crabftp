package log

import (
	"os"
	"time"
	"fmt"
)

var printNormal=true
var printDebug=false
var WTF=false
var logFile *os.File
//StartLog 开启日志
//mode=1 显示普通日志，不显示详细日志，不写入到文件
//mode=2 显示普通日志，显示详细日志，不写入到文件
//mode=3 显示普通日志，不显示详细日志，写入所有日志到文件
//mode=4 显示普通日志，显示详细日志，写入所有日志到文件
func StartLog(mode int){
	os.Mkdir("logfile",0666)
	if mode==1{
	}else if mode==2{
		printDebug=true
	}else if mode==3{
		WTF=true
	}else if mode==4{
		printDebug=true
		WTF=true
	}
	if WTF{
		f,err:=os.OpenFile("logfile/"+time.Now().Format("20060102150405.999.txt"),os.O_CREATE|os.O_APPEND,0666)
		if err!=nil{
			WTF=false
			NPrint("log",err.Error())
		}else{
			logFile=f
		}
	}
}
func NPrint(where string,msg string){
	t:=fmt.Sprintln(time.Now().Format("2006-01-02 15:04:05.999N"),fmt.Sprintf("[%s] %s",where,msg))
	fmt.Print(t)
	if WTF{
		logFile.WriteString(t)
	}
}
func DPrint(where string,msg string){
	if !printDebug{
		return
	}
	t:=fmt.Sprintln(time.Now().Format("2006-01-02 15:04:05.999V"),fmt.Sprintf("[%s] %s",where,msg))
	fmt.Print(t)
	if WTF{
		logFile.WriteString(t)
	}
}