package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strconv"

	adaptix "github.com/Adaptix-Framework/axc2"
)

type Teamserver interface {
	TsAgentIsExists(agentId string) bool
	TsAgentCreate(agentCrc string, agentId string, beat []byte, listenerName string, ExternalIP string, Async bool) (adaptix.AgentData, error)
	TsAgentSetTick(agentId string, listenerName string) error
	TsAgentProcessData(agentId string, bodyData []byte) error
	TsAgentGetHostedAll(agentId string, maxDataSize int) ([]byte, error)
}

type PluginListener struct{}

type Listener struct {
	http *HTTP
}

var (
	ModuleDir       string
	ListenerDataDir string
	Ts              Teamserver
)

func InitPlugin(ts any, moduleDir string, listenerDir string) adaptix.PluginListener {
	ModuleDir = moduleDir
	ListenerDataDir = listenerDir
	Ts = ts.(Teamserver)
	return &PluginListener{}
}

func (p *PluginListener) Create(name string, config string, customData []byte) (adaptix.ExtenderListener, adaptix.ListenerData, []byte, error) {
	var (
		listenerData adaptix.ListenerData
		conf         HTTPConfig
		customdData  []byte
		err          error
	)

	if customData == nil {
		err = json.Unmarshal([]byte(config), &conf)
		if err != nil {
			return nil, listenerData, customdData, err
		}

		randSlice := make([]byte, 16)
		_, _ = rand.Read(randSlice)
		conf.EncryptKey = randSlice[:16]
		conf.Protocol = "http"
	} else {
		err = json.Unmarshal(customData, &conf)
		if err != nil {
			return nil, listenerData, customdData, err
		}
	}

	listener := &HTTP{
		Name:   name,
		Config: conf,
		Active: false,
	}

	listenerData = adaptix.ListenerData{
		BindHost:  listener.Config.HostBind,
		BindPort:  strconv.Itoa(listener.Config.PortBind),
		AgentAddr: conf.Callback_addresses,
		Status:    "Stopped",
	}

	if listener.Config.Ssl {
		listenerData.Protocol = "https"
	}

	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(listener.Config)
	if err != nil {
		return nil, listenerData, customdData, err
	}
	customdData = buffer.Bytes()

	return &Listener{http: listener}, listenerData, customdData, nil
}

func (l *Listener) Start() error {
	return l.http.Start(Ts)
}

func (l *Listener) Edit(config string) (adaptix.ListenerData, []byte, error) {
	var (
		listenerData adaptix.ListenerData
		conf         HTTPConfig
		customdData  []byte
		err          error
	)

	err = json.Unmarshal([]byte(config), &conf)
	if err != nil {
		return listenerData, customdData, err
	}

	l.http.Config.Callback_addresses = conf.Callback_addresses

	listenerData = adaptix.ListenerData{
		BindHost:  l.http.Config.HostBind,
		BindPort:  strconv.Itoa(l.http.Config.PortBind),
		AgentAddr: l.http.Config.Callback_addresses,
		Status:    "Listen",
	}
	if !l.http.Active {
		listenerData.Status = "Closed"
	}

	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(l.http.Config)
	if err != nil {
		return listenerData, customdData, err
	}
	customdData = buffer.Bytes()

	return listenerData, customdData, nil
}

func (l *Listener) Stop() error {
	return l.http.Stop()
}

func (l *Listener) GetProfile() ([]byte, error) {
	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(l.http.Config)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (l *Listener) InternalHandler(data []byte) (string, error) {
	return "", errors.New("not supported")
}
