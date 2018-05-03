package runcmd
import(
	"myutils"
	"regexp"
	"redisdb"
	"strings"
	"io/ioutil"
	"time"
	"path"
	"fmt"
	"strconv"
	"os"
	"io"
	"os/exec"
	"sync"
)
type Data struct {
	 StringData string
	 Mu *sync.Mutex
	 BoolData bool
}
type StepRun struct {
	Sample 	  string
	Prefix 	  string
	Step      string 
	Index     string
	Command   string
	MessageToChan string
	MessageFromChan string
	Db redisdb.Database
	Status    *Data
	SendTicker  *time.Ticker
	HandleTicker *time.Ticker
	ReceiveMess *Data
	MessageQueue *myutils.StringQueue
	DeployId string
	Output *Data
	QueueMutex *sync.Mutex
	FinishedStatus chan string
	BaseDir string
	LogsQueue *myutils.StringQueue
	CmdSet  *myutils.Set
	LogTicker *time.Ticker
	AliveTicker *time.Ticker
}
func NewStepRun() *StepRun{
	redisAddr := myutils.GetOsEnv("REDISADDR")
	db := redisdb.NewRedisDB("tcp",redisAddr)
	sample := myutils.GetOsEnv("SAMPLE")
	prefix := myutils.GetSamplePrefix(sample)
	deployId := myutils.GetOsEnv("DEPLOYMENTID")
	step := myutils.GetOsEnv("STEP")
	index := myutils.GetOsEnv("INDEX")
	infoFrom := myutils.GetOsEnv("RECEIVEMESSAGECHAN")
	infoTo := myutils.GetOsEnv("SENDMESSAGECHAN")
	baseDir := "/mnt/data"
	rcq := myutils.NewStringQueue(120)
	lgsq := myutils.NewStringQueue(1000)
	fstatus := make(chan string)
	stick := time.NewTicker(3 * time.Second)
	logTicker := time.NewTicker(15 * time.Second)
	aliveTicker := time.NewTicker(60 * time.Second)
	return &StepRun{
		Sample: sample,
		Prefix: prefix,
		Step: step,
		Index: index,
		Command: "",
		Output: NewData("ready",false),
		QueueMutex: new(sync.Mutex),
		BaseDir: baseDir,
		MessageToChan: infoTo,
		MessageFromChan: infoFrom,
		Status: NewData("idle",false),
		Db: db,
		CmdSet: myutils.NewSet(),
		DeployId: deployId,
		SendTicker: stick,
		HandleTicker: time.NewTicker(3 * time.Second),
		MessageQueue: rcq,
		LogsQueue: lgsq,
		LogTicker: logTicker,
		AliveTicker: aliveTicker,
		FinishedStatus: fstatus,
		ReceiveMess: NewData("",true)}
	
}
func NewData(str string,bol bool) *Data {
	return &Data{
		Mu: new(sync.Mutex),
		StringData: str,
		BoolData: bol}
}
func (ss *Data) SetStringStatus(data string) {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	ss.StringData = data

}
func (ss *Data) SetBoolStatus(data bool) {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	ss.BoolData = data
}
func (sr *StepRun) StartRun() {
	data := sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + "start"
	sr.DeleteDebugFile()
	go sr.Ticker()
	go sr.FinishedListen()
	if !sr.CheckStatus() {
		sr.HandleData(data)	
	}else {
		_,err := sr.Db.RedisPublish(sr.MessageToChan, sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + "succeed")
		if err != nil {
			sr.AppendLogToQueue("Info","push data error: ",sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + "succeed"," ","reason:",err.Error())
		}
	}
	sr.Db.RedisSubscribe(sr.MessageFromChan,sr.ReceiveData)
}
func (sr *StepRun) SetCommand(cmd string) {
	sr.Command = cmd
}
func (sr *StepRun) ReadCommand(step,index string) string {
	file := path.Join(sr.BaseDir,sr.Sample,step,".command")
	data,err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	info := strings.Split(string(data),"-***-")
	cmd,_ := strconv.Atoi(index)
	if cmd >= len(info) {
		return ""
	}
	return info[cmd]
}
func (sr *StepRun) CheckStatus() bool {
	regstr := sr.Step + "-" + sr.Index + ":.*" + "succeed"
	reg := regexp.MustCompile(regstr)
	file := path.Join(sr.BaseDir,sr.Sample,sr.Step,".status")
	_,err := os.Stat(file)
	if err != nil {
		return false
	}
	data,err := ioutil.ReadFile(file)
	info := string(data)
	if reg.FindString(info) != "" {
		return true
	}
	return false
}
func (sr *StepRun) CmdMessageToSet(info string) bool {
	hash := myutils.GetSha256(info)
	status := sr.CmdSet.Add(hash)
	return status
}
func (sr *StepRun) ReceiveData(data string) {
	go sr.MessageQueue.PushToQueue(data)

}
func (sr *StepRun) HandleData(data string) {
	info := strings.Split(data,":")
	if len(info) != 3 {
		sr.AppendLogToQueue("Error","Get invalid message ",data)
		return
	}
	deployId := info[0]
	status :=  info[2]
	if deployId != sr.DeployId {
		sr.AppendLogToQueue("Info","This deployment id ",deployId," does not match me")
		return 
	}
	if len(strings.Split(info[1],"-")) != 2 {
		sr.AppendLogToQueue("Error","Get invalid message ",data)
		return 
	}
	step := strings.Split(info[1],"-")[0]
	index := strings.Split(info[1],"-")[1]
	if status == "received" {
		if step != sr.Step {
			sr.AppendLogToQueue("Info","This step ",step," does not match me")
			return
		} else if index != sr.Index {
			sr.AppendLogToQueue("Info","This index ",index," does not match me")
			return
		}
		sr.Status.SetStringStatus("idle")
		sr.Output.SetStringStatus("ready")
		sr.ReceiveMess.SetBoolStatus(true)
		return 
	}else if status == "start" {
		if sr.CmdMessageToSet(data) == false {
			return
		}
		_,err := sr.Db.RedisPublish(sr.MessageToChan, deployId + ":" + step + "-" + index + ":" + "received")
		if err != nil {
			sr.AppendLogToQueue("Info","push data error: ",deployId + ":" + step + "-" + index + ":" + "received"," ","reason:",err.Error())
		}
		if sr.Status.StringData == "running" || sr.Output.StringData != "ready" {
			sr.MessageQueue.PushToQueue(data)
			sr.AppendLogToQueue("Info","container is running,the message will be append the message queue again")
			return 
		}else {
			sr.WriteLogs()
			sr.Step = step
			sr.Index = index
			sr.SetCommand(sr.ReadCommand(sr.Step,sr.Index))
			go sr.ExecCmd(step,index)
		} 
	}
}

func (sr *StepRun) FinishedListen() {
	for {
		select {
			case status,ok := <- sr.FinishedStatus:
               	if ok {
                    sr.AppendLogToQueue("Info","get command run status ",status," from channel")
                }else{
                    sr.AppendLogToQueue("Error","channel has closed")
                }   
                sr.Output.SetStringStatus(status)
                _,err := sr.Db.RedisPublish(sr.MessageToChan,sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + sr.Output.StringData)
                if err != nil {
                    sr.AppendLogToQueue("Info",sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + sr.Output.StringData," reason: ",err.Error())
                    
                }
		}
	}

}
func (sr *StepRun) Ticker() {
	for {
		select {
			case <- sr.SendTicker.C:
 				sr.Print()
			case <- sr.LogTicker.C:
  				sr.WriteLogs()
				sr.SendAgain()
			case <- sr.AliveTicker.C:
				sr.KeepAlive()
			case <- sr.HandleTicker.C:
				sr.CheckRun()
				
 		}
	}
}
func (sr *StepRun) KeepAlive() {
	_,err := sr.Db.RedisPublish(sr.MessageToChan,sr.DeployId + ":" + sr.Step + "-" + sr.Index +":" + "alive")
	if err != nil {
		sr.AppendLogToQueue("Info",sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + "alive"," reason: ",err.Error())
	}
}
func (sr *StepRun) Print() {
	printStr := `*********************************************
Current       Time: %s
Run Command Status: %s
Current       Step: %s
Receice Controller: %s
Output      Status: %s
Command     String: %s
*********************************************`
	var bstr string
	if sr.ReceiveMess.BoolData == true {
		bstr = "true"
	}else {
		bstr = "false"
	}
	fmt.Printf(printStr,myutils.GetTime(),sr.Status.StringData,sr.Step,bstr,sr.Output.StringData,sr.Command)
	fmt.Println("\n")
}
func (sr *StepRun) CheckRun() {
	if len(sr.MessageQueue.Queue) != 0 {
		members := sr.MessageQueue.PopAllFromQueue()
		for _,data := range members {
			sr.AppendLogToQueue("Info","pop data from queue")
			sr.HandleData(data)
		}
	}
}
func (sr *StepRun) SendAgain() {
	if sr.Status.StringData == "idle" && sr.Output.StringData != "ready" {
		_,err := sr.Db.RedisPublish(sr.MessageToChan,sr.DeployId + ":" + sr.Step + "-" + sr.Index + ":" + sr.Output.StringData)
		if err != nil {
			 sr.AppendLogToQueue("Info","some error publish: ",err.Error())
		}
		sr.AppendLogToQueue("Info","publish command run status to controller")
	}
}
func (sr *StepRun) ExecCmd(step,index string) {
	defer sr.Status.SetStringStatus("idle")
	defer sr.ReceiveMess.SetBoolStatus(false)
	sr.Status.SetStringStatus("running")
	var status string
	if sr.Command == "" {
		status = "failed"
		sr.AppendLogToQueue("Error","execute command failed,because we don't read the command from .command file")
	}else {
		dir := path.Join(sr.BaseDir,sr.Sample,step)
		os.Chdir(dir)
		outFile := path.Join(dir,"logs",step + "-"  + index + "-output.log")
		errorFile := path.Join(dir,"logs",step + "-" + index + "-error.log")
		status = sr.RunCmd(sr.Command,outFile,errorFile)
		sr.AppendLogToQueue("Info","the command ",sr.Command," will run")
	}
	sr.FinishedStatus <- status
	sr.AppendLogToQueue("Info","the command run status ",sr.Command," has send to channel")
	return 
}
func (sr *StepRun) RunCmd(cmdStr,outlog,errorlog string) (string) {
	_,errSt := os.Stat(errorlog)
    if errSt == nil {
        os.Remove(errorlog)
    }
	cmd := exec.Command("/bin/sh","-c",cmdStr)
	stdout, err := os.OpenFile(outlog, os.O_CREATE|os.O_WRONLY, 0600)
	defer stdout.Close()
	if err != nil {
		sr.AppendLogToQueue("Error","create output log failed,command execute failed,reason: ",err.Error())
		return "failed"
	}
	stderr, err := os.OpenFile(errorlog, os.O_CREATE|os.O_WRONLY, 0600)
    defer stderr.Close()
	if err != nil {
		sr.AppendLogToQueue("Error","create error log failed,command execute failed,reason: ",err.Error())
		return "failed"
	}
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	cmd.Start()
	err3 := cmd.Wait()
	if err3 != nil {
		sr.AppendLogToQueue("Error","command execute error,reason: ",err3.Error())
		return "failed"
	}
	return "succeed" 
}
func (sr *StepRun) AppendLogToQueue(level string,logStr ...string) {
	mystr := myutils.GetTime() 
	mystr = mystr + "\t" + level + "\t"
	for _,val := range logStr {
		mystr = mystr +  val 
	}
	mystr = mystr + "\n"
	sr.LogsQueue.PushToQueue(mystr)	
}
func (sr *StepRun) WriteLogs() {
	data := strings.Join(sr.LogsQueue.PopAllFromQueue(),"")
	logfile := path.Join(sr.BaseDir,sr.Sample,sr.Step,"logs", sr.Step + "-" + sr.Index + "-debug.log")
	var f *os.File
	var err error
	if checkFile(logfile) {
		f,err = os.OpenFile(logfile,os.O_APPEND | os.O_WRONLY,os.ModeAppend)
	}else {
		f,err = os.Create(logfile)
	}
	if err != nil {
		sr.AppendLogToQueue("Error",err.Error())
		return 
	}
	_,err1 := io.WriteString(f,data)
	if err1 != nil {
		sr.AppendLogToQueue("Error",err1.Error())
	}
}
func (sr *StepRun) DeleteDebugFile() {
	debugFile := path.Join(sr.BaseDir,sr.Sample,sr.Step,"logs", sr.Step + "-" + sr.Index + "-debug.log")
	_,err := os.Stat(debugFile)
	if err == nil {
		os.Remove(debugFile)
	}
}
func checkFile(filename string) bool {
	var exist = true
	if _,err := os.Stat(filename);os.IsNotExist(err) {

		exist = false
	}
	return exist
}
