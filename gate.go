package main

import (
	"context"
	"log"

	"go.minekube.com/gate/plugins/cloversecurity"
	"go.minekube.com/gate/plugins/pokehub"
	"go.minekube.com/gate/plugins/pokeskins"

	pokeperms "go.minekube.com/gate/plugins/PokePermsGate"

	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
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