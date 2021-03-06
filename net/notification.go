package loopnet

import (
	"bufio"
	"context"
	"sync"
	// "fmt"
	"log"

	p2p "github.com/acruikshank/loopnet/pb"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

// pattern: /protocol-name/request-or-response-message/version
const notificationRequest = "/loopnet/notify/0.0.1"
const maxNotesPerNotification = 10

// NotificationProtocol type
type NotificationProtocol struct {
	node       *Node      // local host
	NoteStore  *NoteStore // stores all notes
	streams    map[string]inet.Stream
	streamsMux *sync.Mutex
}

func NewNotificationProtocol(node *Node) *NotificationProtocol {
	n := &NotificationProtocol{node: node}
	node.SetStreamHandler(notificationRequest, n.onNotification)
	n.streams = make(map[string]inet.Stream)
	n.streamsMux = &sync.Mutex{}
	return n
}

// remote peer requests handler
func (np *NotificationProtocol) onNotification(s inet.Stream) {
	//log.Printf("%s: Received notification from %s.", np.node.ID(), s.Conn().RemotePeer())
	// get request data
	notification := &p2p.Message{}
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
	err := decoder.Decode(notification)
	if err != nil {
		log.Println(err)
		return
	}

	for _, note := range notification.Notes {
		valid := np.node.authenticateNote(note)

		if !valid {
			log.Println("Failed to authenticate message")
			continue
		}

		if np.NoteStore.OnNote(*note) {
			nodeId, err := peer.IDB58Decode(note.NodeId)
			if err != nil {
				log.Println("Error converting id", err)
				continue
			}

			address, err := ma.NewMultiaddr(note.Address)
			if err != nil {
				log.Println("Error creating address", err)
				continue
			}

			//log.Printf("%s: adding to notes: %s.", np.node.ID(), nodeId)
			np.node.Peerstore().AddAddrs(nodeId, []ma.Multiaddr{address}, ps.PermanentAddrTTL)
		}
	}
}

func (np *NotificationProtocol) Notify() bool {
	destinations := np.NoteStore.RandomNotes(2, true)
	if len(destinations) < 1 {
		// no nodes to notify
		return true
	}

	for _, destination := range destinations {
		nodeId, err := peer.IDB58Decode(destination.NodeId)
		if err != nil {
			log.Println("Error converting id", err)
			return false
		}

		ok := np.sendNotification(nodeId)
		if !ok {
			log.Println("Failed to send")
		}
	}
	return true
}

func (np *NotificationProtocol) ConnectToHost(node *Node) {
	np.node.Peerstore().AddAddrs(node.ID(), node.Addrs(), ps.PermanentAddrTTL)
	np.sendNotification(node.ID())
}

func (np *NotificationProtocol) sendNotification(nodeId peer.ID) bool {
	//log.Printf("%s: Sending notification to %s.", np.node.ID(), nodeId)
	notes := np.NoteStore.RandomNotes(maxNotesPerNotification, false)
	req := &p2p.Message{Notes: notes}

	s, err := np.OpenStream(nodeId)
	if err != nil {
		log.Println("Error opening stream:", err)
		return false
	}

	return np.node.sendProtoMessage(req, s)
}

func (np *NotificationProtocol) OpenStream(nodeId peer.ID) (inet.Stream, error) {
	// np.streamsMux.Lock()
	// defer np.streamsMux.Unlock()
	//
	// s, found := np.streams[nodeId.String()]
	// if found {
	//   return s, nil
	// }
	//
	stream, err := np.node.NewStream(context.Background(), nodeId, notificationRequest)
	if err != nil {
		return nil, err
	}
	// np.streams[nodeId.String()] = stream

	return stream, nil
}
