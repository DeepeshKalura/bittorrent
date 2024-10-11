package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	bencode "github.com/jackpal/bencode-go"
)

var ErrInvalidFormat = errors.New("invalid bencode format")

// Function to decode bencoded data

func parseTorrentFile(filename string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if !strings.HasSuffix(filename, ".torrent") {
		return nil, errors.New("torrent file is invalid")
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	bencodedData := string(fileData)
	torrentData, _, err := decodeBencode(bencodedData)
	if err != nil {
		return nil, err
	}

	if dict, ok := torrentData.(map[string]interface{}); ok {
		if url, found := dict["announce"]; found {
			result["url"] = url
		}

		h := sha1.New()

		h.Write(dict["info"].([]byte))
		result["hash"] = h.Sum(nil)
		if info, found := dict["info"]; found {
			if infoDict, ok := info.(map[string]interface{}); ok {
				result["size"] = infoDict["length"]
			}
		}

	} else {
		return nil, ErrInvalidFormat
	}

	return result, nil
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
		torrentDict, err := parseTorrentFile(os.Args[2])
		if err != nil {
			return
		}
		fmt.Printf("Tracker URL: %s\n", torrentDict["url"])
		fmt.Printf("Length: %d\n", torrentDict["size"])
		fmt.Printf("Info Hash: %x\n", torrentDict["hash"])

	} else {
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
