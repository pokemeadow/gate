package pokeperms

import (
	"encoding/json"
	"fmt"

	"github.com/pokemeadow/gate/pkg/edition/java/proxy"
)

// SyncChannelID implicitly implements Gate's ChannelIdentifier interface
type SyncChannelID string

func (c SyncChannelID) ID() string {
	return string(c)
}

var SyncChannel = SyncChannelID("pokeperms:sync")

type SyncPacket struct {
	Token  string `json:"token"`
	Action string `json:"action"`
	Target string `json:"target"`
}

type MessagingManager struct {
	proxy *proxy.Proxy
	token string
}

func NewMessagingManager(p *proxy.Proxy, token string) *MessagingManager {
	p.ChannelRegistrar().Register(SyncChannel)
	return &MessagingManager{
		proxy: p,
		token: token,
	}
}

func (m *MessagingManager) BroadcastUpdate(action string, target string) error {
	packet := SyncPacket{
		Token:  m.token,
		Action: action,
		Target: target,
	}

	data, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to encode permission packet: %w", err)
	}

	servers := m.proxy.Servers()
	if len(servers) == 0 {
		return nil
	}

	for _, server := range servers {
		players := server.Players()
		var firstConn proxy.ServerConnection

		players.Range(func(p proxy.Player) bool {
			firstConn = p.CurrentServer()
			return false
		})

		if firstConn != nil {
			_ = firstConn.SendPluginMessage(SyncChannel, data)
		}
	}

	return nil
}