package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

var ErrInvalidFormat = errors.New("invalid bencode format")

// Function to decode bencoded data
func decodeBencode(input string) (interface{}, string, error) {
	if len(input) == 0 {
		return nil, "", ErrInvalidFormat
	}

	switch input[0] {
	case 'i':
		return decodeInt(input)
	case 'l':
		return decodeList(input)
	case 'd':
		return decodeDict(input)
	default:
		if input[0] >= '0' && input[0] <= '9' {
			return decodeString(input)
		}
	}

	return nil, "", ErrInvalidFormat
}

func decodeInt(input string) (interface{}, string, error) {
	if input[0] != 'i' {
		return nil, "", ErrInvalidFormat
	}

	endIndex := strings.Index(input, "e")
	if endIndex == -1 {
		return nil, "", ErrInvalidFormat
	}

	numStr := input[1:endIndex]

	// Check for leading zeros
	if len(numStr) > 1 && (numStr[0] == '0' || (numStr[0] == '-' && numStr[1] == '0')) {
		return nil, "", ErrInvalidFormat
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return nil, "", ErrInvalidFormat
	}

	return num, input[endIndex+1:], nil
}

func decodeString(input string) (interface{}, string, error) {
	colonIndex := strings.Index(input, ":")
	if colonIndex == -1 {
		return nil, "", ErrInvalidFormat
	}

	lengthStr := input[:colonIndex]
	length, err := strconv.Atoi(lengthStr)
	if err != nil || length < 0 {
		return nil, "", ErrInvalidFormat
	}

	if len(input) < colonIndex+1+length {
		return nil, "", ErrInvalidFormat
	}

	strValue := input[colonIndex+1 : colonIndex+1+length]
	return strValue, input[colonIndex+1+length:], nil
}

func decodeList(input string) (interface{}, string, error) {
	if input[0] != 'l' {
		return nil, "", ErrInvalidFormat
	}

	input = input[1:]
	var list []interface{}

	for input[0] != 'e' {
		item, rest, err := decodeBencode(input)
		if err != nil {
			return nil, "", err
		}
		list = append(list, item)
		input = rest
	}

	return list, input[1:], nil
}

func decodeDict(input string) (interface{}, string, error) {
	if input[0] != 'd' {
		return nil, "", ErrInvalidFormat
	}

	input = input[1:]
	dict := make(map[string]interface{})
	keys := []string{}

	for input[0] != 'e' {
		key, rest, err := decodeString(input)
		if err != nil {
			return nil, "", err
		}

		strKey, ok := key.(string)
		if !ok {
			return nil, "", ErrInvalidFormat
		}

		value, rest, err := decodeBencode(rest)
		if err != nil {
			return nil, "", err
		}

		dict[strKey] = value
		keys = append(keys, strKey)
		input = rest
	}

	sort.Strings(keys)
	sortedDict := make(map[string]interface{})
	for _, key := range keys {
		sortedDict[key] = dict[key]
	}

	return sortedDict, input[1:], nil
}

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
		if bencodedValue == "le" {
			fmt.Println("[]")
			return
		}
		decode, _, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}
		jsonData, err := json.Marshal(decode)
		if err != nil {
			fmt.Println("[]")
			return
		}
		fmt.Println(string(jsonData))

	} else if command == "info" {
		torrentDict, err := parseTorrentFile(os.Args[2])
		if err != nil {
			return
		}
		fmt.Printf("Tracker URL: %s\n", torrentDict["url"])
		fmt.Printf("Length: %d\n", torrentDict["size"])

	} else {
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
