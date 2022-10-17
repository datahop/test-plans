package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	mRand "math/rand"
	"os"
	"os/user"
	"time"

	"github.com/datahop/ipfs-lite/pkg"
	"github.com/datahop/ipfs-lite/pkg/store"
	"github.com/ipfs/go-cid"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

type Stat struct {
	ProvideBufLen    int
	Wantlist         []cid.Cid
	Peers            []string
	BlocksReceived   uint64
	DataReceived     uint64
	BlocksSent       uint64
	DataSent         uint64
	DupBlksReceived  uint64
	DupDataReceived  uint64
	MessagesReceived uint64
	Latency          uint64
	Time             string
	BandWidth        uint64
}

func newPeer(ctx context.Context, root string) (*pkg.Common, error) {
	absoluteRoot := root + string(os.PathSeparator) + ".datahop"
	err := pkg.Init(absoluteRoot, "4321")
	if err != nil {
		return nil, err
	}
	comm, err := pkg.New(ctx, absoluteRoot, "4321", nil)
	if err != nil {
		return nil, err
	}
	_, err = comm.Start("", true)
	if err != nil {
		return nil, err
	}
	_ = logger.SetLogLevel("replication", "Error")
	return comm, nil
}

func addFile(ctx context.Context, runenv *runtime.RunEnv, tag string, comm *pkg.Common, size int64) error {
	w := &bytes.Buffer{}
	r := &io.LimitedReader{R: rand.Reader, N: size}
	n, err := io.Copy(w, r)
	if err != nil {
		return err
	}
	runenv.RecordMessage("copied %d", n)
	content := bytes.NewReader(w.Bytes())
	info := &store.Info{
		Tag:         tag,
		Type:        "text/plain",
		Name:        "content",
		IsEncrypted: false,
		Size:        content.Size(),
	}
	id, err := comm.Node.Add(ctx, content, info)
	if err != nil {
		return err
	}
	runenv.RecordMessage("Added content %s", id)
	return nil
}

func getFile(ctx context.Context, runenv *runtime.RunEnv, tag string, comm *pkg.Common) error {
	r, info, err := comm.Node.Get(ctx, tag)
	if err != nil {
		return err
	}
	defer r.Close()
	runenv.RecordMessage("Downloading content: %+v", info)
	_, err = io.ReadAll(r)
	return err
}

func printBitswapStat(runenv *runtime.RunEnv, comm *pkg.Common, latency, bandwidth uint64, et time.Duration) error {
	bs, err := comm.Node.BitswapStat()
	if err != nil {
		return err
	}
	obs := &Stat{
		ProvideBufLen:    bs.ProvideBufLen,
		Wantlist:         bs.Wantlist,
		Peers:            bs.Peers,
		BlocksReceived:   bs.BlocksReceived,
		DataReceived:     bs.DataReceived,
		BlocksSent:       bs.BlocksSent,
		DataSent:         bs.DataSent,
		DupBlksReceived:  bs.DupBlksReceived,
		DupDataReceived:  bs.DupDataReceived,
		MessagesReceived: bs.MessagesReceived,
		Latency:          latency,
		Time:             et.String(),
		BandWidth:        bandwidth,
	}
	d, err := json.Marshal(obs)
	if err != nil {
		return err
	}
	runenv.RecordMessage("%s", string(d))
	return nil
}

func BitswapTestCase1(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1Ten(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1Hundred(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1Rand(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1ZeroOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1TenOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1HundredOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1RandOne(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1ZeroTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1TenTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1HundredTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1RandTen(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1ZeroHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1TenHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1HundredHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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

func BitswapTestCase1RandHund(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
		err := addFile(ctx, runenv, tag, comm, oneM)
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
