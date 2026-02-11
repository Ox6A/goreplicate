package main

import (
	"goreplicate/files"
	"goreplicate/networking"
)

func main() {
	networking.StartPeerDiscovery()

	index, err := files.NewFileIndex("files.db")
	if err != nil {
		panic(err)
	}
	defer index.Close()
	err = index.IndexDirectory("/home/localuser/cloneFolder/goreplicate/testfiles")
	if err != nil {
		panic(err)
	}
	select {}
}
