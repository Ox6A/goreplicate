package main

import (
	"fmt"
	"goreplicate/files"
	"goreplicate/networking"
	"os"
	"strconv"
)

func main() {
	// Initialize file index
	index, err := files.NewFileIndex("files.db")
	if err != nil {
		panic(err)
	}
	defer index.Close()

	// Index a directory (use a configurable path or default)
	indexPath := "/home/localuser/cloneFolder/goreplicate/testfiles"
	if len(os.Args) > 1 {
		indexPath = os.Args[1]
	}
	
	// Get TCP port from arguments or use default
	tcpPort := 13583
	if len(os.Args) > 2 {
		tcpPort, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Invalid port number, using default: %d\n", 13583)
			tcpPort = 13583
		}
	}
	
	fmt.Printf("Indexing directory: %s\n", indexPath)
	err = index.IndexDirectory(indexPath)
	if err != nil {
		fmt.Printf("Warning: failed to index directory: %v\n", err)
		// Don't panic, continue with empty index
	}

	// Create file list provider function
	fileListProvider := func() ([]interface{}, error) {
		fileEntries, err := index.GetAllFiles()
		if err != nil {
			return nil, err
		}
		
		// Convert to []interface{} for JSON encoding
		result := make([]interface{}, len(fileEntries))
		for i, entry := range fileEntries {
			result[i] = entry
		}
		return result, nil
	}

	// Start P2P networking with file list provider
	err = networking.StartPeerDiscoveryWithP2P(fileListProvider, tcpPort)
	if err != nil {
		panic(err)
	}

	fmt.Println("P2P file sharing started. Discovering peers and sharing file lists...")
	fmt.Printf("Usage: %s [directory_path] [tcp_port]\n", os.Args[0])
	fmt.Println("Press Ctrl+C to exit")
	
	// Keep running
	select {}
}
