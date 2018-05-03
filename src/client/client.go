package client
import (
	"fmt"
	"bufio"
	"redisdb"
	"validator"
	"kube"
	"os"
	"sync"
	"cpfile"
	"time"
	//"encoding/json"
	"myutils"
	"strings"
	"strconv"
	"path"
	"syscall"
	"os/signal"
)
type Client struct {
	Db redisdb.Database
	Sample string
	Prefix string
	InputDir string
	BaseDir string
	TmpTemplate string
	TemplateName string
	RunningTaskSet string
	SendMessageChan string
	ReceiveMessageChan string
	Template string
	Cover string
	ControllerContainer string
	DelConSignal  chan os.Signal
	MyTicker  *time.Ticker
	AliveTicker *time.Ticker
	PipeObj   *validator.Validator  
	StartTime time.Time
	StepLock  *sync.RWMutex
	StepStatusInfo string
	IsRunning bool
	Kube *kube.K8sClient
	LastAliveTime time.Time
	FailedCount int
	CheckAlive bool
}
/*
func main() {
	data,_ := ioutil.ReadFile("pipelineTest.json")
	cli := NewClient("10.61.0.86:6379","yang2","/mnt/data","yang1",string(data),"true",make(map[string]string))
	cli.Start()
}
*/
func NewClient(redisAddr,inputDir,glusterDir,sample,template,cover,tname,istmp string,params map[string]string) *Client {
	db := redisdb.NewRedisDB("tcp",redisAddr)
	k8s := kube.NewK8sClient(redisAddr)
	prefix := myutils.GetSamplePrefix(sample)
	indir := inputDir
	referDir := "/mnt/refer"
	dataDir := "/mnt/data"
	rts := "Kubernetes-Running-Sample-Set"
	smc := prefix + "-Client-To-Controller"
	rmc := prefix + "-Listen-From-Controller"
	tk := time.NewTicker(5 * time.Second)
	lc  := new(sync.RWMutex)
	sticker := time.NewTicker(15 * time.Second)
	va,err := validator.NewValidator(template,referDir,path.Join(dataDir,sample),params)
	if err != nil {
		myutils.Print("Error",err.Error(),true)
	}
	va.StartValidate()
	return &Client {
		Db: db,
		TemplateName: tname,
		TmpTemplate: istmp,
		Sample: sample,
		Prefix: prefix,
		InputDir: indir,
		StepStatusInfo: "",
		BaseDir: path.Join(glusterDir,sample),
		RunningTaskSet: rts,
		Cover: cover,
		StepLock: lc,
		Kube: k8s,
		StartTime: time.Now(),
		DelConSignal: make(chan os.Signal,1),
		AliveTicker: sticker,
		Template: template,
		ReceiveMessageChan: rmc,
		PipeObj: va,
		FailedCount: 0,
		CheckAlive: false,
		IsRunning: false,
		SendMessageChan: smc,
		MyTicker: tk}
}
func (cli *Client) Start() {
	go cli.ListenInterrupt()
	if cli.CheckSampleIsRunning() == true {
		cli.HandleSample()
	}else {
		cli.Init()
	}
	go cli.RoundToGetStatus()
	go cli.PrintStatus()
	cli.Db.RedisSubscribe(cli.ReceiveMessageChan,cli.Listen)
}
func (cli *Client) HandleSample() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("the sample has been exists,do you want to delete it?[yes/no]: ")
	rdata,_,_ := reader.ReadLine()
	read := string(rdata)
	if read == "yes" {
		if cli.Kube.DeploymentExist(cli.Prefix) != 1 {
        	_,err := cli.Kube.DeleteDeployment(cli.Prefix)
        	if err != nil {
           		myutils.Print("Error","delete the error controller failed,reason: " + err.Error(),true)
        	}   
        	time.Sleep(30 * time.Second)
    	} 
		cli.Db.RedisSetSremMember(cli.RunningTaskSet,cli.Sample)
       	os.Exit(0)
	}else if read == "no" {
		return
	}else {
            fmt.Printf("Error: you only can input \"yes\" or \"no\"\n")
            os.Exit(3)
	}
}
func (cli *Client) Init() {
	cli.Db.RedisSetAdd(cli.RunningTaskSet,cli.Sample)
	cli.CopyFile()
	cli.RunAllStepsAgain()
	if cli.Kube.DeploymentExist(cli.Prefix) != 1 {
		_,err := cli.Kube.DeleteDeployment(cli.Prefix)
		if err != nil {
			myutils.Print("Error","delete the error controller failed,reason: " + err.Error(),true)
		}   
		myutils.Print("Info","waiting kubeletes to delete the pre error status controller",false)
		time.Sleep(45 * time.Second)
    } 
	_,err := cli.Kube.CreateController(cli.Sample)
	if err != nil {
		myutils.Print("Error","create controller failed,reason: " + err.Error(),true)
	}
	cli.LastAliveTime = time.Now()
}
func (cli *Client) CheckSampleIsRunning() bool  {
	status,err := cli.Db.RedisSetSisMember(cli.RunningTaskSet,cli.Sample)
	redata := true
	if err != nil {
		myutils.Print("Error","read status of sample " + cli.Sample + " from redis failed,reason: " + err.Error(),true)
	}
	if !status {
		redata = false
	}
	return redata
}
func (cli *Client) CheckContrAlive() {
	dur := time.Since(cli.LastAliveTime)
	timeout := time.Duration(80) * time.Second
	if dur > timeout && !cli.CheckAlive && cli.FailedCount < 4 {
		cli.FailedCount++
	    if cli.Kube.DeploymentExist(cli.Prefix) != 1 {
        	_,err := cli.Kube.DeleteDeployment(cli.Prefix)
        	if err != nil {
            	myutils.Print("Error","delete the error controller failed,reason: " + err.Error(),false)
        	}   
        	myutils.Print("Info","waiting kubeletes to delete the pre error status controller",false)
        	time.Sleep(60 * time.Second)
    	} 
    	_,err := cli.Kube.CreateController(cli.Sample)
    	if err != nil {
        	myutils.Print("Error","create controller failed,reason: " + err.Error(),false)
    	}
    	cli.LastAliveTime = time.Now()
		cli.CheckAlive = true
	}else if cli.FailedCount >= 4 {
	    if cli.Kube.DeploymentExist(cli.Prefix) != 1 {
        	_,err := cli.Kube.DeleteDeployment(cli.Prefix)
        	if err != nil {
            	myutils.Print("Error","delete the error controller failed,reason: " + err.Error(),false)
        	} 
			myutils.Print("Error","recreate the controller failed with three time,exit",true)  
		}
	}
}
func (cli *Client) ListenInterrupt() {
	signal.Notify(cli.DelConSignal,syscall.SIGHUP,syscall.SIGINT,syscall.SIGTERM,syscall.SIGTSTP)
	for {
		select {
			case <- cli.DelConSignal:
				cli.Db.RedisPublish(cli.SendMessageChan,cli.Prefix + "###" + "delete")
				myutils.Print("Error","you cancel the task",true)
		}
	
	}
}
func (cli *Client) RoundToGetStatus() {
	for {
		select {
			case <- cli.MyTicker.C:
				cli.GetStepsStatus()
			case <- cli.AliveTicker.C:
				cli.CheckContrAlive()
		}
	}

}
func (cli *Client) Listen(data string) {
	info := strings.Split(data,"###")
	if len(info) == 2 {
		prefix  := info[0]
		status := info[1]
		if prefix != cli.Prefix {
			return 
		}else if status != "alive" {
			return 
		}else {
			cli.LastAliveTime = time.Now()
		}
	}
} 
func (cli *Client) GetStepsStatus() {
	data,err := cli.Db.RedisStringGet(cli.Prefix)
	if err != nil {
		return 
	}
	name,tm := cli.GetTempTime()
	data = strings.Replace(data,"TEMPLATE",name,1)
	data = strings.Replace(data,"ESTIMATE",tm,1)
	data = strings.Replace(data,"HAVARUNTIME",myutils.GetRunTime(cli.StartTime),1)
	cli.StepStatusInfo = data
}
func (cli *Client) PrintInfo() {
	if cli.StepStatusInfo == "" {
		myutils.Print("Info","no step info to get,waiting",false)
		return
	}
	info := strings.Split(cli.StepStatusInfo,"\n")
	for _,val := range info {
		fmt.Println(val)
		time.Sleep(600 * time.Millisecond)
	}
	fmt.Println("\n")
}
func (cli *Client) PrintStatus() {
	for {
		cli.PrintInfo()
		time.Sleep(1 * time.Second)
		if cli.CheckSampleIsRunning() == false  {
			cli.GetStepsStatus()
			cli.PrintInfo()
    		myutils.Print("Info","the task has ran finished,bye!",false)
        	os.Exit(0)
		}
	}
}
func (cli *Client) CopyFile() {
	os.MkdirAll(path.Join(cli.BaseDir,"step0"),0755)
	for key,_ := range cli.PipeObj.NormData {
		os.MkdirAll(path.Join(cli.BaseDir,key,"logs"),0755)
	}
	cpfile.CopyFilesToGluster(path.Join(cli.BaseDir,"step0"),cli.InputDir,cli.Template)
	os.Remove(path.Join(cli.BaseDir,"step0",".template"))
	cli.PipeObj.WriteObjToFile(path.Join(cli.BaseDir,"step0","pipeline.json"))
	if cli.TmpTemplate == "false" {
		myutils.WriteFile(path.Join(cli.BaseDir,"step0",".template"),cli.TemplateName,true)
	}
}
func (cli *Client) RunAllStepsAgain() {
	if cli.Cover == "true" {
		for key,_ := range cli.PipeObj.NormData {
			os.Remove(path.Join(cli.BaseDir,key,".status"))
		}
	}

}
func (cli *Client) GetTempTime() (string,string) {
	if cli.TmpTemplate == "false" {
		pipeid := "pipeid" + myutils.GetSha256(strings.Trim(cli.TemplateName," "))[:15]
        tm,err := cli.Db.RedisHashGet(pipeid,"estimate-time")
		if err  == nil {
			tmInt,_ := strconv.ParseInt(tm,10,64)
			return cli.TemplateName,myutils.GetRunTimeWithSeconds(tmInt)
		}
		tm,err = cli.Db.RedisHashGet(cli.TemplateName,"estimate-time")
		if err != nil {
			return "get template failed","0h 0m 0s"
		}
		name,_ := cli.Db.RedisHashGet(cli.TemplateName,"pipeline-name")
		tmInt,_ := strconv.ParseInt(tm,10,64)
       	return name,myutils.GetRunTimeWithSeconds(tmInt)

	}else {

		return "temporary template","0h 0m 0s"
	}


}
func NormString(info string,length int) string {
	llen := len(info)
	if llen <= length {
		return info + strings.Repeat(" ",length - llen) 
	}else {
		return info[0:length]
	}

}
