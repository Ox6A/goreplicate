package networking

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type PeerConnection struct {
	conn       net.Conn
	identifier string
	address    string
}

type FileList struct {
	Identifier string        `json:"identifier"`
	Files      []interface{} `json:"files"` // Using interface{} to accept any file structure
	Timestamp  time.Time     `json:"timestamp"`
}

type P2PServer struct {
	fileListProvider func() ([]interface{}, error)
	mutex            sync.Mutex
	connections      map[string]*PeerConnection
	tcpPort          int
}

// NewP2PServer creates a new P2P server with a file list provider function
func NewP2PServer(fileListProvider func() ([]interface{}, error), tcpPort int) *P2PServer {
	return &P2PServer{
		fileListProvider: fileListProvider,
		connections:      make(map[string]*PeerConnection),
		tcpPort:          tcpPort,
	}
}

// StartServer starts the TCP server to accept incoming connections
func (s *P2PServer) StartServer(identifier string) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.tcpPort))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %v", err)
	}
	fmt.Printf("P2P server listening on port %d\n", s.tcpPort)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}
			fmt.Printf("Accepted connection from %s\n", conn.RemoteAddr().String())
			go s.handleConnection(conn, identifier)
		}
	}()

	return nil
}

// handleConnection handles an incoming peer connection
func (s *P2PServer) handleConnection(conn net.Conn, myIdentifier string) {
	defer conn.Close()

	// Read the request from the peer
	decoder := json.NewDecoder(conn)
	var request map[string]string
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Printf("Error decoding request: %v\n", err)
		return
	}

	peerIdentifier := request["identifier"]
	action := request["action"]

	fmt.Printf("Received request from peer %s: %s\n", peerIdentifier, action)

	if action == "get_file_list" {
		// Get file list from provider
		files, err := s.fileListProvider()
		if err != nil {
			fmt.Printf("Error getting file list: %v\n", err)
			return
		}

		// Create response
		response := FileList{
			Identifier: myIdentifier,
			Files:      files,
			Timestamp:  time.Now(),
		}

		// Send response
		encoder := json.NewEncoder(conn)
		err = encoder.Encode(response)
		if err != nil {
			fmt.Printf("Error encoding response: %v\n", err)
			return
		}

		fmt.Printf("Sent file list to peer %s (%d files)\n", peerIdentifier, len(files))
	}
}

// ConnectToPeer connects to a peer and requests their file list
func (s *P2PServer) ConnectToPeer(identifier, peerAddress, myIdentifier string, peerPort int) (*FileList, error) {
	// Extract IP address without port
	host, _, err := net.SplitHostPort(peerAddress)
	if err != nil {
		// If SplitHostPort fails, assume it's just an IP
		host = peerAddress
	}

	// Connect to peer's TCP server
	address := fmt.Sprintf("%s:%d", host, peerPort)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %v", address, err)
	}
	defer conn.Close()

	// Send request
	request := map[string]string{
		"identifier": myIdentifier,
		"action":     "get_file_list",
	}

	encoder := json.NewEncoder(conn)
	err = encoder.Encode(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var fileList FileList
	err = decoder.Decode(&fileList)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed by peer")
		}
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Printf("Received file list from peer %s (%d files)\n", fileList.Identifier, len(fileList.Files))
	return &fileList, nil
}

// RequestFileListFromPeer is a convenience method to request file lists from discovered peers
func (s *P2PServer) RequestFileListFromPeer(peer Peer, myIdentifier string) {
	fileList, err := s.ConnectToPeer(peer.identifier, peer.address, myIdentifier, peer.tcpPort)
	if err != nil {
		fmt.Printf("Error requesting file list from peer %s: %v\n", peer.identifier, err)
		return
	}

	fmt.Printf("Successfully retrieved file list from peer %s at %s\n", fileList.Identifier, fileList.Timestamp.Format(time.RFC3339))
	fmt.Printf("Files received: %d\n", len(fileList.Files))
	
	// Print first few files as sample
	for i, file := range fileList.Files {
		if i >= 5 {
			fmt.Printf("... and %d more files\n", len(fileList.Files)-5)
			break
		}
		fileJSON, _ := json.MarshalIndent(file, "", "  ")
		fmt.Printf("File %d: %s\n", i+1, string(fileJSON))
	}
}
