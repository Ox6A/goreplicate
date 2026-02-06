# P2P File List Sharing - Implementation Summary

## Overview
This implementation adds peer-to-peer (P2P) file list sharing functionality to goreplicate, enabling multiple peers to discover each other and exchange JSON-encoded file lists in a decentralized manner, similar to Syncthing.

## Architecture

### Components

1. **File Indexing** (`files/files.go`)
   - SQLite-based file metadata storage
   - SHA256 hashing for file integrity
   - JSON-serializable file entries
   - `GetAllFiles()` method to retrieve complete file inventory

2. **Peer Discovery** (`networking/discovery.go`)
   - UDP broadcast on port 13582 for peer discovery
   - Automatic peer detection and registration
   - Integration with P2P connections

3. **P2P Connections** (`networking/peer.go`)
   - TCP-based peer-to-peer connections
   - Request/response protocol for file list exchange
   - JSON encoding for data transmission
   - Configurable TCP ports (default: 13583)

### Communication Protocol

#### Discovery Phase
- Each peer broadcasts UDP messages: `replicationbroadcast:<identifier>:<tcp_port>`
- Peers listen on port 13582 for broadcasts
- When a new peer is discovered, an automatic P2P connection is established

#### File List Exchange
1. **Request**: JSON message with `{"identifier": "...", "action": "get_file_list"}`
2. **Response**: JSON message with:
   ```json
   {
     "identifier": "peer_id",
     "files": [
       {
         "path": "/path/to/file",
         "hash": "sha256_hash",
         "size": 1234,
         "mod_time": "2026-02-06T12:00:00Z",
         "is_dir": false
       }
     ],
     "timestamp": "2026-02-06T12:00:00Z"
   }
   ```

## Usage

### Running a Single Peer
```bash
./goreplicate [directory_path] [tcp_port]
```

### Running Multiple Peers
```bash
# Peer 1
./goreplicate /path/to/folder1 13583

# Peer 2 (in another terminal)
./goreplicate /path/to/folder2 13584
```

## Features

- ✅ Peer-to-peer architecture (no central server)
- ✅ Automatic peer discovery via UDP broadcasts
- ✅ JSON-encoded file list exchange
- ✅ Support for multiple peers
- ✅ Configurable TCP ports for running multiple instances
- ✅ File metadata including hashes, sizes, and timestamps
- ✅ Directory structure support

## Testing

The implementation has been tested with:
- 2 peers running on the same machine
- Different TCP ports to avoid conflicts
- File list exchange verification
- JSON encoding/decoding validation

## Future Enhancements

Potential improvements for future development:
- Actual file synchronization (currently only shares file lists)
- Conflict resolution strategies
- Incremental updates (only send changed files)
- Encryption for secure communication
- Authentication between peers
- Network-wide peer discovery (currently limited to localhost)
- Persistent peer relationships
- File transfer progress tracking

## Security Considerations

- Currently uses unencrypted TCP connections
- No authentication mechanism
- Trusts all discovered peers
- File paths are shared as-is (could expose directory structure)

For production use, additional security measures should be implemented.
