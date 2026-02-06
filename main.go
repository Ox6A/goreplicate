package main

import "goreplicate/networking"

func main() {
	networking.StartPeerDiscovery()
	select {}
}
