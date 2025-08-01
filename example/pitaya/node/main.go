package main

import (
	"strings"

	"github.com/topfreegames/pitaya/v3/examples/demo/cluster/services"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/pkg/groups"
)

var app pitaya.Pitaya

func main() {
	cfg := config.NewDefaultPitayaConfig()
	cfg.Cluster.SD.Etcd.Endpoints = []string{"43.199.45.25:2379"}
	cfg.Cluster.RPC.Server.Nats.Connect = "43.199.45.25:4222"
	cfg.Cluster.RPC.Client.Nats.Connect = "43.199.45.25:4222"
	builder := pitaya.NewDefaultBuilder(false, "room", pitaya.Cluster, map[string]string{}, *cfg)
	builder.Groups = groups.NewMemoryGroupService(builder.Config.Groups.Memory)
	app = builder.Build()
	defer app.Shutdown()

	room := services.NewRoom(app)
	app.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	app.RegisterRemote(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	app.Start()
}
