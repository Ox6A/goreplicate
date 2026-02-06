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
	tcpPort    int
}

const (
	port          int    = 13582
	messagePrefix string = "replicationbroadcast"
	baseTCPPort   int    = 13583
)

var (
	message    string
	identifier string
	peers      Peers
	wg         sync.WaitGroup
	p2pServer  *P2PServer
)

func concatenateMessage(identifier string, tcpPort int) string {
	return fmt.Sprintf("%s:%s:%d", messagePrefix, identifier, tcpPort)
}

func newPeer(identifier, address string, tcpPort int) Peer {
	peers.mutex.Lock()
	defer peers.mutex.Unlock()
	for i, peer := range peers.Peers {
		if peer.identifier == identifier {
			peers.Peers[i].lastSeen = time.Now()
			return peers.Peers[i]
		}
	}
	peer := Peer{
		identifier: identifier,
		address:    address,
		lastSeen:   time.Now(),
		tcpPort:    tcpPort,
	}
	peers.Peers = append(peers.Peers, peer)
	return peer
}

func udpBroadcast(identifier string, tcpPort int) {
	fmt.Printf("udpBroadcast: starting (identifier=%s, tcp_port=%d)\n", identifier, tcpPort)
	address, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		fmt.Printf("udpBroadcast: FATAL - failed to resolve UDP address: %v\n", err)
		panic(fmt.Sprintf("UDP broadcast initialization failed: %v", err))
	}
	fmt.Printf("udpBroadcast: resolved address %s\n", address.String())
	connection, err := net.DialUDP("udp4", nil, address)
	if err != nil {
		fmt.Printf("udpBroadcast: FATAL - failed to dial UDP: %v\n", err)
		panic(fmt.Sprintf("UDP broadcast connection failed: %v", err))
	}
	fmt.Printf("udpBroadcast: connection established\n")
	defer connection.Close()
	for {
		msg := concatenateMessage(identifier, tcpPort)
		n, err := connection.Write([]byte(msg))
		if err != nil {
			fmt.Printf("udpBroadcast: write error: %v\n", err)
			// Don't panic for write errors, just log and continue
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Printf("udpBroadcast: sent %d bytes: %s\n", n, msg)
		time.Sleep(5 * time.Second)
	}
}

func udpListen(identifier string, tcpPort int) {
	fmt.Printf("udpListen: starting (identifier=%s, tcp_port=%d)\n", identifier, tcpPort)
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
		fmt.Printf("udpListen: FATAL - ListenPacket error: %v\n", err)
		panic(fmt.Sprintf("UDP listener initialization failed: %v", err))
	}
	fmt.Printf("udpListen: listening on port %d\n", port)
	defer packetConnection.Close()
	var buffer [1024]byte
	for {
		receivedBytes, address, err := packetConnection.ReadFrom(buffer[:])
		if err != nil {
			fmt.Printf("udpListen: ReadFrom error: %v\n", err)
			continue
		}
		fmt.Printf("Received message: %s from %s\n", string(buffer[:receivedBytes]), address.String())
		receivedMessage := string(buffer[0:receivedBytes])
		receivedParts := strings.Split(receivedMessage, ":")
		if len(receivedParts) == 3 && receivedParts[0] == messagePrefix && receivedMessage != concatenateMessage(identifier, tcpPort) {
			fmt.Printf("Received valid message with identifier: %s, tcp_port: %s\n", receivedParts[1], receivedParts[2])
			
			// Parse TCP port
			var peerTCPPort int
			_, err := fmt.Sscanf(receivedParts[2], "%d", &peerTCPPort)
			if err != nil || peerTCPPort <= 0 {
				fmt.Printf("Warning: failed to parse TCP port from peer message: %v\n", err)
				continue
			}
			
			peer := newPeer(receivedParts[1], address.String(), peerTCPPort)
			fmt.Printf("Added new peer: %s at %s (TCP port: %d)\n", receivedParts[1], address.String(), peerTCPPort)
			for _, p := range peers.Peers {
				fmt.Printf("Known peer: %s at %s (TCP port: %d, last seen: %s)\n", p.identifier, p.address, p.tcpPort, p.lastSeen.Format(time.RFC3339))
			}
			
			// Connect to peer and request file list if P2P server is initialized
			if p2pServer != nil {
				go p2pServer.RequestFileListFromPeer(peer, identifier)
			}
		}
	}
}

func StartPeerDiscovery() {
	identifier = fmt.Sprintf("%d", rand.Int())
	fmt.Printf("StartPeerDiscovery: generated identifier=%s\n", identifier)
	fmt.Printf("StartPeerDiscovery: launching discovery goroutines\n")
	go udpListen(identifier, baseTCPPort)
	go udpBroadcast(identifier, baseTCPPort)
}

// StartPeerDiscoveryWithP2P starts peer discovery and P2P server
func StartPeerDiscoveryWithP2P(fileListProvider func() ([]interface{}, error), tcpPort int) error {
	identifier = fmt.Sprintf("%d", rand.Int())
	fmt.Printf("StartPeerDiscoveryWithP2P: generated identifier=%s\n", identifier)
	
	// Initialize P2P server
	p2pServer = NewP2PServer(fileListProvider, tcpPort)
	err := p2pServer.StartServer(identifier)
	if err != nil {
		return fmt.Errorf("failed to start P2P server: %v", err)
	}
	
	fmt.Printf("StartPeerDiscoveryWithP2P: launching discovery goroutines\n")
	go udpListen(identifier, tcpPort)
	go udpBroadcast(identifier, tcpPort)
	
	return nil
}
