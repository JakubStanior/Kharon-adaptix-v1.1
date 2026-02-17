package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	adaptix "github.com/Adaptix-Framework/axc2"
)

const (
	OS_UNKNOWN = 0
	OS_WINDOWS = 1
	OS_LINUX   = 2
	OS_MAC     = 3

	TYPE_TASK       = 1
	TYPE_BROWSER    = 2
	TYPE_JOB        = 3
	TYPE_TUNNEL     = 4
	TYPE_PROXY_DATA = 5

	MESSAGE_INFO    = 5
	MESSAGE_ERROR   = 6
	MESSAGE_SUCCESS = 7

	DOWNLOAD_STATE_RUNNING  = 1
	DOWNLOAD_STATE_STOPPED  = 2
	DOWNLOAD_STATE_FINISHED = 3
	DOWNLOAD_STATE_CANCELED = 4
)

type Teamserver interface {
	TsListenerInteralHandler(watermark string, data []byte) (string, error)

	TsAgentProcessData(agentId string, bodyData []byte) error

	TsAgentUpdateData(newAgentData adaptix.AgentData) error
	TsAgentTerminate(agentId string, terminateTaskId string) error

	TsAgentConsoleOutput(agentId string, messageType int, message string, clearText string, store bool)
	TsAgentConsoleOutputClient(agentId string, client string, messageType int, message string, clearText string)

	TsPivotCreate(pivotId string, pAgentId string, chAgentId string, pivotName string, isRestore bool) error
	TsGetPivotInfoByName(pivotName string) (string, string, string)
	TsGetPivotInfoById(pivotId string) (string, string, string)
	TsPivotDelete(pivotId string) error

	TsTaskCreate(agentId string, cmdline string, client string, taskData adaptix.TaskData)
	TsTaskUpdate(agentId string, data adaptix.TaskData)
	TsTaskGetAvailableAll(agentId string, availableSize int) ([]adaptix.TaskData, error)

	TsDownloadAdd(agentId string, fileId string, fileName string, fileSize int) error
	TsDownloadUpdate(fileId string, state int, data []byte) error
	TsDownloadClose(fileId string, reason int) error
	TsDownloadDelete(fileId []string) error
	TsScreenshotAdd(agentId string, Note string, Content []byte) error
	TsClientGuiDisksWindows(taskData adaptix.TaskData, drives []adaptix.ListingDrivesDataWin)
	TsClientGuiFiles(taskData adaptix.TaskData, path string, jsonFiles string)
	TsClientGuiFilesStatus(taskData adaptix.TaskData)
	TsClientGuiProcess(taskData adaptix.TaskData, jsonFiles string)

	TsTunnelStart(TunnelId string) (string, error)
	TsTunnelCreateSocks4(AgentId string, Info string, Lhost string, Lport int) (string, error)
	TsTunnelCreateSocks5(AgentId string, Info string, Lhost string, Lport int, UseAuth bool, Username string, Password string) (string, error)
	TsTunnelCreateLportfwd(AgentId string, Info string, Lhost string, Lport int, Thost string, Tport int) (string, error)
	TsTunnelCreateRportfwd(AgentId string, Info string, Lport int, Thost string, Tport int) (string, error)
	TsTunnelUpdateRportfwd(tunnelId int, result bool) (string, string, error)

	TsTunnelStopSocks(AgentId string, Port int)
	TsTunnelStopLportfwd(AgentId string, Port int)
	TsTunnelStopRportfwd(AgentId string, Port int)

	TsTunnelConnectionClose(channelId int, writeOnly bool)
	TsTunnelConnectionResume(AgentId string, channelId int, ioDirect bool)
	TsTunnelConnectionData(channelId int, data []byte)
	TsTunnelConnectionAccept(tunnelId int, channelId int)
}

type PluginAgent struct{}

type ExtenderAgent struct{}

var (
	Ts             Teamserver
	ModuleDir      string
	AgentWatermark string
)

func InitPlugin(ts any, moduleDir string, watermark string) adaptix.PluginAgent {
	ModuleDir = moduleDir
	AgentWatermark = watermark
	Ts = ts.(Teamserver)
	return &PluginAgent{}
}

func (p *PluginAgent) GetExtender() adaptix.ExtenderAgent {
	return &ExtenderAgent{}
}

/// PluginAgent — build/create

func (p *PluginAgent) GenerateProfiles(profile adaptix.BuildProfile) ([][]byte, error) {
	var agentProfiles [][]byte

	for _, transportProfile := range profile.ListenerProfiles {
		var listenerMap map[string]any
		if err := json.Unmarshal(transportProfile.Profile, &listenerMap); err != nil {
			return nil, err
		}

		agentProfile, err := AgentGenerateProfile(profile.AgentConfig, transportProfile.Watermark, listenerMap)
		if err != nil {
			return nil, err
		}
		agentProfiles = append(agentProfiles, agentProfile)
	}
	return agentProfiles, nil
}

func (p *PluginAgent) BuildPayload(profile adaptix.BuildProfile, agentProfiles [][]byte) ([]byte, string, error) {
	if len(profile.ListenerProfiles) != 1 || len(agentProfiles) != 1 {
		return nil, "", errors.New("only one listener profile is supported")
	}

	var listenerMap map[string]any
	if err := json.Unmarshal(profile.ListenerProfiles[0].Profile, &listenerMap); err != nil {
		return nil, "", err
	}

	return AgentGenerateBuild(profile.AgentConfig, agentProfiles[0], listenerMap)
}

func (p *PluginAgent) CreateAgent(beat []byte) (adaptix.AgentData, adaptix.ExtenderAgent, error) {
	agentData, err := CreateAgent(beat)
	if err != nil {
		return agentData, nil, err
	}
	return agentData, &ExtenderAgent{}, nil
}

/// ExtenderAgent — runtime

func (ext *ExtenderAgent) Encrypt(data []byte, key []byte) ([]byte, error) {
	return AgentEncryptData(data, key)
}

func (ext *ExtenderAgent) Decrypt(data []byte, key []byte) ([]byte, error) {
	return AgentDecryptData(data, key)
}

func (ext *ExtenderAgent) PackTasks(agentData adaptix.AgentData, tasks []adaptix.TaskData) ([]byte, error) {
	return PackTasks(agentData, tasks)
}

func (ext *ExtenderAgent) PivotPackData(pivotId string, data []byte) (adaptix.TaskData, error) {
	packData, err := PackPivotTasks(pivotId, data)
	if err != nil {
		return adaptix.TaskData{}, err
	}

	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	uid := hex.EncodeToString(randomBytes)[:8]

	return adaptix.TaskData{
		TaskId: uid,
		Type:   TYPE_PROXY_DATA,
		Data:   packData,
		Sync:   false,
	}, nil
}

func (ext *ExtenderAgent) CreateCommand(agentData adaptix.AgentData, args map[string]any) (adaptix.TaskData, adaptix.ConsoleMessageData, error) {
	return CreateTask(Ts, agentData, args)
}

func (ext *ExtenderAgent) ProcessData(agentData adaptix.AgentData, decryptedData []byte) error {
	taskData := adaptix.TaskData{
		Type:        TYPE_TASK,
		AgentId:     agentData.Id,
		FinishDate:  time.Now().Unix(),
		MessageType: MESSAGE_SUCCESS,
		Completed:   true,
		Sync:        true,
	}

	resultTasks := ProcessTasksResult(Ts, agentData, taskData, decryptedData)
	for _, task := range resultTasks {
		Ts.TsTaskUpdate(agentData.Id, task)
	}
	return nil
}

/// TUNNEL

func (ext *ExtenderAgent) TunnelCallbacks() adaptix.TunnelCallbacks {
	return adaptix.TunnelCallbacks{
		ConnectTCP: TunnelMessageConnectTCP,
		ConnectUDP: TunnelMessageConnectUDP,
		WriteTCP:   TunnelMessageWriteTCP,
		WriteUDP:   TunnelMessageWriteUDP,
		Close:      TunnelMessageClose,
		Reverse:    TunnelMessageReverse,
	}
}

func TunnelMessageConnectTCP(channelId int, tunnelType int, addressType int, address string, port int) adaptix.TaskData {
	packData, _ := TunnelCreateTCP(channelId, address, port)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TunnelMessageConnectUDP(channelId int, tunnelType int, addressType int, address string, port int) adaptix.TaskData {
	packData, _ := TunnelCreateUDP(channelId, address, port)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TunnelMessageWriteTCP(channelId int, data []byte) adaptix.TaskData {
	packData, _ := TunnelWriteTCP(channelId, data)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TunnelMessageWriteUDP(channelId int, data []byte) adaptix.TaskData {
	packData, _ := TunnelWriteUDP(channelId, data)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TunnelMessageClose(channelId int) adaptix.TaskData {
	packData, _ := TunnelClose(channelId)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TunnelMessageReverse(tunnelId int, port int) adaptix.TaskData {
	packData, _ := TunnelReverse(tunnelId, port)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

/// TERMINAL

func (ext *ExtenderAgent) TerminalCallbacks() adaptix.TerminalCallbacks {
	return adaptix.TerminalCallbacks{
		Start: TerminalMessageStart,
		Write: TerminalMessageWrite,
		Close: TerminalMessageClose,
	}
}

func TerminalMessageStart(terminalId int, program string, sizeH int, sizeW int, oemCP int) adaptix.TaskData {
	packData, _ := TerminalStart(terminalId, program, sizeH, sizeW)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TerminalMessageWrite(terminalId int, oemCP int, data []byte) adaptix.TaskData {
	packData, _ := TerminalWrite(terminalId, data)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func TerminalMessageClose(terminalId int) adaptix.TaskData {
	packData, _ := TerminalClose(terminalId)
	return adaptix.TaskData{Type: TYPE_PROXY_DATA, Data: packData, Sync: false}
}

/// SYNC

func SyncBrowserDisks(ts Teamserver, taskData adaptix.TaskData, drivesSlice []adaptix.ListingDrivesDataWin) {
	ts.TsClientGuiDisksWindows(taskData, drivesSlice)
}

func SyncBrowserFiles(ts Teamserver, taskData adaptix.TaskData, path string, filesSlice []adaptix.ListingFileDataWin) {
	jsonFiles, err := json.Marshal(filesSlice)
	if err != nil {
		return
	}
	ts.TsClientGuiFiles(taskData, path, string(jsonFiles))
}

func SyncBrowserFilesStatus(ts Teamserver, taskData adaptix.TaskData) {
	ts.TsClientGuiFilesStatus(taskData)
}

func SyncBrowserProcess(ts Teamserver, taskData adaptix.TaskData, processlist []adaptix.ListingProcessDataWin) {
	jsonProcess, err := json.Marshal(processlist)
	if err != nil {
		return
	}
	ts.TsClientGuiProcess(taskData, string(jsonProcess))
}
