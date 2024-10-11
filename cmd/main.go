package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	bencode "github.com/jackpal/bencode-go"
)

type MetaInfo struct {
	Name        string
	Pieces      string
	Length      int64
	PieceLength int64
}
type Meta struct {
	Announce string
	Info     MetaInfo
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: program <command> <argument>")
		return
	}
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]
		data, err := bencode.Decode(strings.NewReader(bencodedValue))

		if err != nil {
			panic(err)
		}
		jsonData, err := json.Marshal(data)

		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsonData))

	} else if command == "info" {

		fileName := os.Args[2]
		f, err := os.ReadFile(fileName)
		if err != nil {
			panic(err)
		}

		var meta Meta
		err = bencode.Unmarshal(bytes.NewReader(f), &meta)
		if err != nil {
			panic(err)

		}
		h := sha1.New()

		bencode.Marshal(h, meta.Info)

		fmt.Println("Tracker URL:", meta.Announce)
		fmt.Println("Length:", meta.Info.Length)
		fmt.Printf("Info Hash: %x\n", h.Sum(nil))

	} else {
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
