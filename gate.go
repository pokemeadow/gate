package main

import (
    "go.minekube.com/gate/plugins/cloversecurity"
    "go.minekube.com/gate/plugins/pokehub"
    "go.minekube.com/gate/plugins/pokeskins"

    "go.minekube.com/gate/cmd/gate"
    "go.minekube.com/gate/pkg/edition/java/proxy"
)

func main() {
    proxy.Plugins = append(proxy.Plugins,
       cloversecurity.Plugin,
       pokehub.Plugin,
       pokeskins.Plugin,
    )

    gate.Execute()
}