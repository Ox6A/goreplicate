package networking

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Peers struct {
	Peers []Peer
	mutex sync.Mutex
}

type Peer struct {
	identifier string
	address    string
	lastSeen   time.Time
}

const (
	port          int    = 13582
	messagePrefix string = "replicationbroadcast"
)

var (
	message    string
	identifier string
	peers      Peers
	wg         sync.WaitGroup
)

func concatenateMessage(identifier string) string {
	return fmt.Sprintf("%s:%s", messagePrefix, identifier)
}

func newPeer(identifier, address string) Peer {
	peers.mutex.Lock()
	defer peers.mutex.Unlock()
	for _, peer := range peers.Peers {
		if peer.identifier == identifier {
			peer.lastSeen = time.Now()
			return peer
		}
	}
	peer := Peer{
		identifier: identifier,
		address:    address,
		lastSeen:   time.Now(),
	}
	peers.Peers = append(peers.Peers, peer)
	return peer
}

func udpBroadcast(identifier string) {
	fmt.Printf("udpBroadcast: starting (identifier=%s)\n", identifier)
	address, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		fmt.Printf("udpBroadcast: failed to resolve UDP address: %v\n", err)
		panic(err)
	}
	fmt.Printf("udpBroadcast: resolved address %s\n", address.String())
	connection, err := net.DialUDP("udp4", nil, address)
	if err != nil {
		fmt.Printf("udpBroadcast: failed to dial UDP: %v\n", err)
		panic(err)
	}
	fmt.Printf("udpBroadcast: connection established\n")
	defer connection.Close()
	for {
		msg := concatenateMessage(identifier)
		n, err := connection.Write([]byte(msg))
		if err != nil {
			fmt.Printf("udpBroadcast: write error: %v\n", err)
			panic(err)
		}
		fmt.Printf("udpBroadcast: sent %d bytes: %s\n", n, msg)
		time.Sleep(5 * time.Second)
	}
}

func udpListen(identifier string) {
	fmt.Printf("udpListen: starting (identifier=%s)\n", identifier)
	listenConfig := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			c.Control(func(fd uintptr) {
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if err != nil {
					return
				}
			})
			return err
		},
	}
	fmt.Printf("udpListen: binding to port %d\n", port)
	packetConnection, err := listenConfig.ListenPacket(context.TODO(), "udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("udpListen: ListenPacket error: %v\n", err)
		panic(err)
	}
	fmt.Printf("udpListen: listening on port %d\n", port)
	defer packetConnection.Close()
	var buffer [1024]byte
	for {
		receivedBytes, address, err := packetConnection.ReadFrom(buffer[:])
		if err != nil {
			fmt.Printf("udpListen: ReadFrom error: %v\n", err)
			panic(err)
		}
		fmt.Printf("Received message: %s from %s\n", string(buffer[:receivedBytes]), address.String())
		receivedMessage := string(buffer[0:receivedBytes])
		receivedParts := strings.Split(receivedMessage, ":")
		if len(receivedParts) == 2 && receivedParts[0] == messagePrefix && receivedMessage != concatenateMessage(identifier) {
			fmt.Printf("Received valid message with identifier: %s\n", receivedParts[1])
			newPeer(receivedParts[1], address.String())
			fmt.Printf("Added new peer: %s at %s\n", receivedParts[1], address.String())
			for _, peer := range peers.Peers {
				fmt.Printf("Known peer: %s at %s (last seen: %s)\n", peer.identifier, peer.address, peer.lastSeen.Format(time.RFC3339))
			}
		}
	}
}

func StartPeerDiscovery() {
	identifier = fmt.Sprintf("%d", rand.Int())
	fmt.Printf("StartPeerDiscovery: generated identifier=%s\n", identifier)
	fmt.Printf("StartPeerDiscovery: launching discovery goroutines\n")
	go udpListen(identifier)
	go udpBroadcast(identifier)
}
