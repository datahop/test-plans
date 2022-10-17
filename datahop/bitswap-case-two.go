package main

import (
	"context"
	"log"
	mRand "math/rand"
	"os/user"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func BitswapTestCase2(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, 0, 0, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2Ten(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency: ten * time.Millisecond,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, ten, 0, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2Hundred(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency: hundred * time.Millisecond,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, hundred, 0, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2Rand(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	mRand.Seed(time.Now().UnixNano())
	r := mRand.Intn(max-ten+1) + ten
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency: time.Duration(r) * time.Millisecond,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, uint64(r), 0, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2ZeroOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Bandwidth: oneMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}
	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Minute)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, 0, oneMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2TenOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   ten * time.Millisecond,
			Bandwidth: oneMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, ten, oneMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2HundredOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   hundred * time.Millisecond,
			Bandwidth: oneMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, hundred, oneMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2RandOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	mRand.Seed(time.Now().UnixNano())
	r := mRand.Intn(max-ten+1) + ten
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   time.Duration(r) * time.Millisecond,
			Bandwidth: oneMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, uint64(r), oneMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2ZeroTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Bandwidth: tenMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}
	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, 0, tenMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2TenTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   ten * time.Millisecond,
			Bandwidth: tenMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, ten, tenMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2HundredTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   hundred * time.Millisecond,
			Bandwidth: tenMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, hundred, tenMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2RandTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	mRand.Seed(time.Now().UnixNano())
	r := mRand.Intn(max-ten+1) + ten
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   time.Duration(r) * time.Millisecond,
			Bandwidth: tenMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, uint64(r), tenMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2ZeroHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Bandwidth: hundredMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}
	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, 0, hundredMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2TenHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   ten * time.Millisecond,
			Bandwidth: hundredMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, ten, hundredMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2HundredHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   hundred * time.Millisecond,
			Bandwidth: hundredMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, hundred, hundredMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func BitswapTestCase2RandHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	mRand.Seed(time.Now().UnixNano())
	r := mRand.Intn(max-ten+1) + ten
	totalTime := time.Minute * 60
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	netclient := network.NewClient(client, runenv)
	config := &network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   time.Duration(r) * time.Millisecond,
			Bandwidth: hundredMbit,
		},
		CallbackState: "network-configured",
	}

	err = netclient.ConfigureNetwork(ctx, config)
	if err != nil {
		return err
	}

	comm, err := newPeer(ctx, usr.HomeDir)
	if err != nil {
		return err
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	if seq == 1 {
		_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
		client.Subscribe(ctx, st, tch)
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	for {
		runenv.RecordMessage("Peer count now %d", len(comm.Node.Peers()))
		if len(comm.Node.Peers()) == runenv.TestInstanceCount-1 {
			initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
			break
		}
		<-time.After(time.Second * 2)
	}
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))

	tag := "tag"
	if seq == 1 {
		err := addFile(ctx, runenv, tag, comm, tenM)
		if err != nil {
			log.Fatal(err)
		}
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "content added", runenv.TestInstanceCount)
	<-time.After(time.Second * 5)
	now := time.Now()
	if seq != 1 {
		err = getFile(ctx, runenv, tag, comm)
		if err != nil {
			log.Fatal(err)
		}
	}
	et := time.Since(now)
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	err = printBitswapStat(runenv, comm, uint64(r), hundredMbit, et)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
