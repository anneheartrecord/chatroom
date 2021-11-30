package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)
//创建用户结构体类型
type Client struct {
	C chan string
	Name string
	Address string
}
//创建全局map 存储在线用户
var onlineMap map[string]Client

//创建全局channel 用来传递用户消息
var message =make(chan string)

func MakeMsg(clnt Client,msg string) (buf string) {
	buf="["+clnt.Address+"]"+clnt.Name+":"+msg
	return
}
func HandlerConnect(conn net.Conn)  {
	defer conn.Close()
    //创建channel判断用户是否活跃
    hasData:=make(chan bool)
	//获取用户ip支持 （客户端） .String() 强转为string类型变量
	netAddr:=conn.RemoteAddr().String()
    //创建新链接用户  默认用户名是ip+port
    clnt:=Client{make(chan string),netAddr,netAddr}
    //将新链接用户添加到在线用户map中
    onlineMap[netAddr]=clnt

    //创建专门用来给当前用户发送消息的go程
    go WriteMsgToClient(clnt,conn)

    //发送用户上线消息到msg通道
    message<-MakeMsg(clnt,"login")
    //创建一个channel 用来判断用户退出状态
    isQuit:=make(chan bool)

    //创建一个匿名Go程，处理用户发送的消息
    go func() {
    	for {
    		buf:=make([]byte,128)
    		n,err:=conn.Read(buf)
    		if n==0{
                isQuit<-true
    			fmt.Printf("检测到客户端:%s退出\n",clnt.Name)
    			return
			}
			if err!=nil {
				fmt.Println("conn read err",err)
				return
			}

			//将读到的用户消息，写到msg中
			msg:=string(buf[:n-1])

			//提取在线用户列表
			if msg=="who"&&len(msg)==3{
				conn.Write([]byte("online user list:\n"))
				//遍历当前map 获取在线用户
				for _,user:=range onlineMap{
					userInfo:=user.Address+":"+user.Name+"\n"
					conn.Write([]byte(userInfo))
				}//判断用户发送了改名命令
			} else if len(msg)>=8&&msg[:6]=="rename" {  //rename |
                newName:=strings.Split(msg,"|")[1]  //msg[8:]
                clnt.Name=newName  //修改结构体成员name
                onlineMap[netAddr]=clnt //	更新在线用户列表
                conn.Write([]byte("rename successful\n"))
			} else {
				message<-MakeMsg(clnt,msg)
			}
            hasData<-true
		}
	}()
    //保证不退出
    for {
           //监听channel的数据流动
    	select {
    	   case <-isQuit:
    	   	    delete(onlineMap,clnt.Address)
    	   	    message<-MakeMsg(clnt,"logout")  //写入用户退出消息到全局channel
    	   	    return
    	   case <-time.After(time.Second*10):
    	   	delete(onlineMap,clnt.Address)
    	   	message<-MakeMsg(clnt,"logout")
    	   	return
    	   case <-hasData:
    	   	//目的是重置case计时器
		}
	}
}
func WriteMsgToClient(clnt Client,conn net.Conn)  {
	//监听用户自带channel里是否有消息
	for msg:=range clnt.C {
		conn.Write([]byte(msg+"\n"))
	}
}
func Manager()  {
	//初始化onlineMap
	onlineMap=make(map[string]Client)
	//循环从msg里面读消息
    for {
		//监听全局channel中是否有数据  有数据就存储至msg 无数据则阻塞
		msg:=<-message

		//循环发送消息给所有在线用户
		for _,value:=range onlineMap {
              value.C<-msg
		}
	}

}
func main()  {
	//创建监听socket
	listen,err:=net.Listen("tcp","127.0.0.1:8080")
	if err!=nil {
		fmt.Println("listen err:",err)
		return
	}
	defer listen.Close()  //记得要defer关闭
	//创建管理者Go程 管理map和全局channel
	go Manager()
	for{
		conn,err:=listen.Accept()  //for循环 accept()监听客户端请求
		if err!=nil {
			fmt.Println("accept err",err)
			return
		}
		//启动go程处理客户端请求
		go HandlerConnect(conn)
	}

}
