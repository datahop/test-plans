package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/datahop/ipfs-lite/pkg"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

var testCases = map[string]interface{}{
	"connection":         Connection,
	"private-connection": PrivateConnection,
	"group-open":         GroupOpen,
	"group-closed":       GroupClosed,
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
	_, err = client.Publish(ctx, st, networkPeers{
		AddrInfo: comm.Node.AddrInfo(),
		Seq:      seq,
	})
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

type networkPeers struct {
	AddrInfo *peer.AddrInfo
	Seq      int64
}

func GroupOpen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 1
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)
	if !runenv.TestSidecar {
		return fmt.Errorf("env does not support sidecar")
	}
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
		},
		CallbackState: "network-configured",
	}

	err := netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}
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

	st := sync.NewTopic("transfer-addr", &networkPeers{})
	_, err = client.Publish(ctx, st, &networkPeers{
		AddrInfo: comm.Node.AddrInfo(),
		Seq:      seq,
	})
	if err != nil {
		panic(err)
	}

	tch := make(chan *networkPeers, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	allPeers := map[int64]*peer.AddrInfo{}
	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.AddrInfo.ID != comm.Node.AddrInfo().ID {
			allPeers[t.Seq] = t.AddrInfo
			err = comm.Node.Connect(*t.AddrInfo)
			if err != nil {
				panic(err)
			}
		}
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	m := comm.Node.ReplManager()
	if seq == 1 {
		gm, err := m.CreateOpenGroup("TestGroup", comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
		if err != nil {
			panic(err)
		}
		runenv.RecordMessage("Peer %s Created group: %s", comm.Node.AddrInfo().ID, gm.GroupID.String())

		nemMemberPubKey := comm.Node.GetPubKey(allPeers[2].ID)
		err = m.GroupAddMember(comm.Node.AddrInfo().ID, allPeers[2].ID, gm.GroupID, comm.Node.GetPrivKey(), nemMemberPubKey)
		if err != nil {
			return err
		}
	}

	<-time.After(time.Second * 5)
	if seq == 2 {
		groups, err := m.GroupGetAllGroups(comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			runenv.RecordMessage("group id: %s, name: %s, owner: %s", groups[0].GroupID.String(), groups[0].Name, groups[0].OwnerID.String())
		} else {
			return fmt.Errorf("peer 2 is not a member of the created group")
		}
		for _, addrInfo := range allPeers {
			nemMemberPubKey := comm.Node.GetPubKey(addrInfo.ID)
			err = m.GroupAddMember(comm.Node.AddrInfo().ID, addrInfo.ID, groups[0].GroupID, comm.Node.GetPrivKey(), nemMemberPubKey)
			if err != nil {
				return err
			}
		}
	}

	<-time.After(time.Second * 5)
	groups, err := m.GroupGetAllGroups(comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
	if err != nil {
		return err
	}
	runenv.RecordMessage("Peer %d is a member of %d group", seq, len(groups))
	if len(groups) > 0 {
		runenv.RecordMessage("group id: %s, name: %s, owner: %s", groups[0].GroupID.String(), groups[0].Name, groups[0].OwnerID.String())
	}
	return nil
}

func GroupClosed(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 1
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)
	if !runenv.TestSidecar {
		return fmt.Errorf("env does not support sidecar")
	}
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
		},
		CallbackState: "network-configured",
	}

	err := netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}
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

	st := sync.NewTopic("transfer-addr", &networkPeers{})
	_, err = client.Publish(ctx, st, &networkPeers{
		AddrInfo: comm.Node.AddrInfo(),
		Seq:      seq,
	})
	if err != nil {
		panic(err)
	}

	tch := make(chan *networkPeers, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	allPeers := map[int64]*peer.AddrInfo{}
	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.AddrInfo.ID != comm.Node.AddrInfo().ID {
			allPeers[t.Seq] = t.AddrInfo
			err = comm.Node.Connect(*t.AddrInfo)
			if err != nil {
				panic(err)
			}
		}
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	m := comm.Node.ReplManager()
	if seq == 1 {
		gm, err := m.CreateGroup("TestGroup", comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
		if err != nil {
			panic(err)
		}
		runenv.RecordMessage("Peer %s Created group: %s", comm.Node.AddrInfo().ID, gm.GroupID.String())

		nemMemberPubKey := comm.Node.GetPubKey(allPeers[2].ID)
		err = m.GroupAddMember(comm.Node.AddrInfo().ID, allPeers[2].ID, gm.GroupID, comm.Node.GetPrivKey(), nemMemberPubKey)
		if err != nil {
			return err
		}
	}

	<-time.After(time.Second * 5)
	if seq == 2 {
		groups, err := m.GroupGetAllGroups(comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			runenv.RecordMessage("group id: %s, name: %s, owner: %s", groups[0].GroupID.String(), groups[0].Name, groups[0].OwnerID.String())
		} else {
			return fmt.Errorf("peer 2 is not a member of the created group")
		}
		for _, addrInfo := range allPeers {
			nemMemberPubKey := comm.Node.GetPubKey(addrInfo.ID)
			
			err = m.GroupAddMember(comm.Node.AddrInfo().ID, addrInfo.ID, groups[0].GroupID, comm.Node.GetPrivKey(), nemMemberPubKey)
			if err != nil {
				return err
			}
		}
	}

	<-time.After(time.Second * 5)
	groups, err := m.GroupGetAllGroups(comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
	if err != nil {
		return err
	}
	runenv.RecordMessage("Peer %d is a member of %d group", seq, len(groups))
	if len(groups) > 0 {
		runenv.RecordMessage("group id: %s, name: %s, owner: %s", groups[0].GroupID.String(), groups[0].Name, groups[0].OwnerID.String())
	}
	return nil
}
