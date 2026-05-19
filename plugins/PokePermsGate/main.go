package pokeperms

import (
	"context"
	"fmt"

	"go.minekube.com/gate/pkg/edition/java/proxy"
)

// PluginName and Version metadata variables
const (
	PluginName    = "PokePermsGate"
	PluginVersion = "1.0.0"
)

type PokePermsGate struct {
	proxy    *proxy.Proxy
	config   *Config
	storage  *Storage
	msgMgr   *MessagingManager
	token    string
}

func NewPlugin(p *proxy.Proxy) *PokePermsGate {
	return &PokePermsGate{
		proxy: p,
	}
}

func (p *PokePermsGate) Init(ctx context.Context) error {
	dir := fmt.Sprintf("plugins/%s", PluginName)

	cfg, err := LoadConfig(dir)
	if err != nil {
		return fmt.Errorf("[%s] Failed to load config.yml: %w", PluginName, err)
	}
	p.config = cfg

	token, err := LoadOrGenerateToken(dir)
	if err != nil {
		return fmt.Errorf("[%s] Failed to load/generate token.yml: %w", PluginName, err)
	}
	p.token = token

	storage, err := NewStorage(dir, cfg)
	if err != nil {
		return fmt.Errorf("[%s] Failed to initialize database driver storage: %w", PluginName, err)
	}
	p.storage = storage

	p.msgMgr = NewMessagingManager(p.proxy, p.token)

	RegisterCommands(p.proxy, p.storage, p.config, p.msgMgr)

	fmt.Printf("[%s] Successfully loaded version %s! Database Driver Connected: %s\n",
		PluginName, PluginVersion, cfg.StorageType)
	fmt.Printf("[%s] Security Verification Token initialized. Ready to sync with PokePerms Paper!\n", PluginName)

	return nil
}

func (p *PokePermsGate) Disable() {
	if p.storage != nil {
		p.storage.Close()
		fmt.Printf("[%s] Database connections successfully closed and safely locked.\n", PluginName)
	}
	fmt.Printf("[%s] Plugin successfully disabled.\n", PluginName)
}