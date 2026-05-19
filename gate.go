package main

import (
	"context"
	"log"

	"github.com/pokemeadow/gate/plugins/cloversecurity"
	"github.com/pokemeadow/gate/plugins/pokehub"
	"github.com/pokemeadow/gate/plugins/pokeskins"

	pokeperms "github.com/pokemeadow/gate/plugins/PokePermsGate"

	"github.com/pokemeadow/gate/cmd/gate"
	"github.com/pokemeadow/gate/pkg/edition/java/proxy"
)

func main() {
	var pokePermsPlugin *pokeperms.PokePermsGate

	pokePermsHook := proxy.Plugin{
		Name: "PokePermsGate",
		Init: func(ctx context.Context, p *proxy.Proxy) error {
			log.Println("[PokePermsGate] Hooking into Gate lifecycle...")

			pokePermsPlugin = pokeperms.NewPlugin(p)

			if err := pokePermsPlugin.Init(ctx); err != nil {
				return err
			}

			log.Println("[PokePermsGate] Plugin successfully injected and operational.")
			return nil
		},
	}

	proxy.Plugins = append(proxy.Plugins,
		cloversecurity.Plugin,
		pokehub.Plugin,
		pokeskins.Plugin,
		pokePermsHook,
	)

	gate.Execute()
}