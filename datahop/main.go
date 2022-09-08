package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"time"

	"github.com/datahop/ipfs-lite/pkg"
	"github.com/datahop/ipfs-lite/pkg/store"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

const (
	// file sizes
	oneM     = 1024 * 1024
	tenM     = oneM * 10
	twentyM  = tenM * 2
	fiftyM   = tenM * 5
	hundredM = fiftyM * 2

	// bandwidth
	oneMbit     = 125000
	tenMbit     = oneMbit * 10
	hundredMbit = tenMbit * 10

	// latency
	ten     = 10
	hundred = ten * 10
	max     = hundred * 2
)

var testCases = map[string]interface{}{
	"connection":           Connection,
	"private-connection":   PrivateConnection,
	"group":                Group,
	"content-distribution": ContentDistribution,

	"bitswap1":      BitswapTestCase1,
	"bitswap1-10":   BitswapTestCase1Ten,
	"bitswap1-100":  BitswapTestCase1Hundred,
	"bitswap1-rand": BitswapTestCase1Rand,

	"bitswap1-0-one":    BitswapTestCase1ZeroOne,
	"bitswap1-10-one":   BitswapTestCase1TenOne,
	"bitswap1-100-one":  BitswapTestCase1HundredOne,
	"bitswap1-rand-one": BitswapTestCase1RandOne,

	"bitswap1-0-ten":    BitswapTestCase1ZeroTen,
	"bitswap1-10-ten":   BitswapTestCase1TenTen,
	"bitswap1-100-ten":  BitswapTestCase1HundredTen,
	"bitswap1-rand-ten": BitswapTestCase1RandTen,

	"bitswap1-0-hund":    BitswapTestCase1ZeroHund,
	"bitswap1-10-hund":   BitswapTestCase1TenHund,
	"bitswap1-100-hund":  BitswapTestCase1HundredHund,
	"bitswap1-rand-hund": BitswapTestCase1RandHund,

	"bitswap2":      BitswapTestCase2,
	"bitswap2-10":   BitswapTestCase2Ten,
	"bitswap2-100":  BitswapTestCase2Hundred,
	"bitswap2-rand": BitswapTestCase2Rand,

	"bitswap2-0-one":    BitswapTestCase2ZeroOne,
	"bitswap2-10-one":   BitswapTestCase2TenOne,
	"bitswap2-100-one":  BitswapTestCase2HundredOne,
	"bitswap2-rand-one": BitswapTestCase2RandOne,

	"bitswap2-0-ten":    BitswapTestCase2ZeroTen,
	"bitswap2-10-ten":   BitswapTestCase2TenTen,
	"bitswap2-100-ten":  BitswapTestCase2HundredTen,
	"bitswap2-rand-ten": BitswapTestCase2RandTen,

	"bitswap2-0-hund":    BitswapTestCase2ZeroHund,
	"bitswap2-10-hund":   BitswapTestCase2TenHund,
	"bitswap2-100-hund":  BitswapTestCase2HundredHund,
	"bitswap2-rand-hund": BitswapTestCase2RandHund,
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
		log.Fatal(err)
	}
	comm, err := pkg.New(context.Background(), absoluteRoot, "4321", nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = comm.Start("", false)
	if err != nil {
		log.Fatal(err)
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
	if err != nil {
		log.Fatal(err)
	}
	tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
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
		log.Fatal(err)
	}
	comm, err := pkg.New(context.Background(), absoluteRoot, "4321", nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = comm.Start("PrivateConnection", false)
	if err != nil {
		log.Fatal(err)
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()
	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	runenv.RecordMessage("Listen address %s for node %d", comm.Node.AddrInfo(), seq)
	_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
	if err != nil {
		log.Fatal(err)
	}
	tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			runenv.RecordMessage("Trying to Connect with %s", t)
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	runenv.RecordMessage("Peer count %d", len(comm.Node.Peers()))
	runenv.RecordMessage("Peers %s", comm.Node.Peers())
	return nil
}

func Group(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
			Latency:   250 * time.Millisecond,
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
		log.Fatal(err)
	}
	comm, err := pkg.New(context.Background(), absoluteRoot, "4321", nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = comm.Start("", false)
	if err != nil {
		log.Fatal(err)
	}

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)

	st := sync.NewTopic("transfer-addr", &peer.AddrInfo{})
	_, err = client.Publish(ctx, st, comm.Node.AddrInfo())
	if err != nil {
		log.Fatal(err)
	}
	tch := make(chan *peer.AddrInfo, runenv.TestInstanceCount)
	client.Subscribe(ctx, st, tch)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		t := <-tch
		if t.ID != comm.Node.AddrInfo().ID {
			err = comm.Node.Connect(*t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	m := comm.Node.ReplManager()
	if seq == 1 {
		gm, err := m.CreateOpenGroup("TestGroup", comm.Node.AddrInfo().ID, comm.Node.GetPrivKey())
		if err != nil {
			log.Fatal(err)
		}
		runenv.RecordMessage("Peer %s Created group: %s", comm.Node.AddrInfo().ID, gm.GroupID.String())

		peers := comm.Node.Peers()
		for _, peerID := range peers {
			newPeerId, err := peer.Decode(peerID)
			if err != nil {
				return err
			}
			nemMemberPubKey := comm.Node.GetPubKey(newPeerId)
			err = m.GroupAddMember(comm.Node.AddrInfo().ID, newPeerId, gm.GroupID, comm.Node.GetPrivKey(), nemMemberPubKey)
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

func ContentDistribution(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	totalTime := time.Minute * 5
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()
	initCtx.MustWaitAllInstancesInitialized(ctx)

	seq := initCtx.SyncClient.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
	usr, err := user.Current()
	absoluteRoot := usr.HomeDir + string(os.PathSeparator) + ".datahop"
	err = pkg.Init(absoluteRoot, "4321")
	if err != nil {
		log.Fatal(err)
	}
	comm, err := pkg.New(ctx, absoluteRoot, "4321", nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = comm.Start("", true)
	if err != nil {
		log.Fatal(err)
	}
	_ = logger.SetLogLevel("replication", "Error")

	initCtx.SyncClient.MustSignalAndWait(ctx, "listening", runenv.TestInstanceCount)
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

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
	contentTopic := sync.NewTopic("transfer-content-addr", "")
	cch := make(chan string, runenv.TestInstanceCount)
	client.Subscribe(ctx, contentTopic, cch)
	tag := "tag"
	if seq == 1 {
		content := bytes.NewReader([]byte("some_content"))
		info := &store.Info{
			Tag:         tag,
			Type:        "text/plain",
			Name:        "content",
			IsEncrypted: false,
			Size:        content.Size(),
		}
		id, err := comm.Node.Add(ctx, content, info)
		if err != nil {
			log.Fatal(err)
		}
		_, err = client.Publish(ctx, contentTopic, tag)
		if err != nil {
			log.Fatal(err)
		}
		runenv.RecordMessage("Added content %s", id)
	}
	<-time.After(time.Second * 10)
	tags, err := comm.Node.ReplManager().GetAllTags()
	if err != nil {
		log.Fatal(err)
	}
	runenv.RecordMessage("tags %v", tags)

	if seq != 1 {
		tag := <-cch
		runenv.RecordMessage("Downloading content with tag: %s", tag)
		r, info, err := comm.Node.Get(ctx, tag)
		if err != nil {
			log.Fatal(err)
		}
		defer r.Close()
		runenv.RecordMessage("Downloaded content: %+v", info)
		data, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}
		runenv.RecordMessage("Got data  %s", string(data))
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "download completed", runenv.TestInstanceCount)
	bs, err := comm.Node.BitswapStat()
	if err != nil {
		log.Fatal(err)
	}
	runenv.RecordMessage("Bitswap Stat for node %d : %+v", seq, bs)
	cancel()
	return nil
}
