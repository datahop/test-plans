package main

import (
	"context"
	"os"
	"os/user"
	"time"

	"github.com/datahop/ipfs-lite/pkg"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

var testCases = map[string]interface{}{
	"connection":         Connection,
	"private-connection": PrivateConnection,
}

func main() {
	run.InvokeMap(testCases)
}

func Connection(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 1
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	absoluteRoot := usr.HomeDir + string(os.PathSeparator) + ".datahop"
	err = pkg.Init(absoluteRoot, "4321")
	if err != nil {
		panic(err)
	}
	comm, err := pkg.New(context.Background(), absoluteRoot, "4321", nil)
	if err != nil {
		panic(err)
	}
	_, err = comm.Start("", false)
	if err != nil {
		panic(err)
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
	if err != nil {
		panic(err)
	}
	tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				panic(err)
			}
		}
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))
	runenv.RecordMessage("Peers %s", comm.Node.Peers())
	return nil
}

func PrivateConnection(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	runenv.RecordMessage("running instance count", runenv.TestInstanceCount)

	totalTime := time.Minute * 1
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	absoluteRoot := usr.HomeDir + string(os.PathSeparator) + ".datahop"
	err = pkg.Init(absoluteRoot, "4321")
	if err != nil {
		panic(err)
	}
	comm, err := pkg.New(context.Background(), absoluteRoot, "4321", nil)
	if err != nil {
		panic(err)
	}
	_, err = comm.Start("PrivateConnection", false)
	if err != nil {
		panic(err)
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
	if err != nil {
		panic(err)
	}
	tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				panic(err)
			}
		}
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))
	runenv.RecordMessage("Peers %s", comm.Node.Peers())
	return nil
}
