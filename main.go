package main

import (
	"context"
	"fmt"
  "time"
	// "log"
	"math/rand"

  loopnet "github.com/acruikshank/loopnet/net"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	swarm "gx/ipfs/QmSwZMWwFZSUpe5muU2xgTUwppH24KfMwdPXiwbEp2c6G5/go-libp2p-swarm"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	ps "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

// helper method - create a lib-p2p host to listen on a port
func createNode(note int) *loopnet.Node {
	// Ignoring most errors for brevity
	// See echo example for more details and better implementation
	port := rand.Intn(1000) + 10000
	priv, pub, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	pid, _ := peer.IDFromPublicKey(pub)
	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
  if err != nil {
    panic("Could not create multiaddress")
  }
  fmt.Println("Created multiaddress")
	peerStore := ps.NewPeerstore()
	peerStore.AddPrivKey(pid, priv)
	peerStore.AddPubKey(pid, pub)
	n, err := swarm.NewNetwork(context.Background(), []ma.Multiaddr{listen}, pid, peerStore, nil)
  if err != nil {
    panic(err)
  }

	host := bhost.New(n)

	node := loopnet.NewNode(host)
  noteData := node.NewNoteData(0,note,false)
  node.NotificationProtocol.NoteStore = loopnet.NewNoteStore(noteData)
  return node
}

// TODO:
// Create goroutine to update note revision and notify
// Create goroutine to clear dead notes
// Add CLI
// Add UI and synthesis

func main() {
  // Choose random ports between 10000-10100
	rand.Seed(666)

	// Make 10 nodes
  nodes := make([]*loopnet.Node,0)
  for i := 0; i < 5; i++ {
    nodes = append(nodes, createNode(60+i))
  }

  // connect round robin
  for i, node := range nodes {
    node.ConnectToHost(nodes[(i+1)%len(nodes)])
  }

  // run 10 rounds of notifications
  for i := 0; i < 100; i++ {
    for j, node := range nodes {
      fmt.Printf("Having %v (%d) notify\n", node.ID(), j)
      node.Notify()
      time.Sleep(1000*time.Millisecond)
    }

    fmt.Println("\n\n\n\nRound", i+1)
    for j, node := range nodes {
      fmt.Printf("Host %d: %v\n", j+1, node.NoteStore.ActiveNoteNumbers())
    }
  }
}
