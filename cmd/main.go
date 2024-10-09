package main

import (
	"fmt"
	"os"
	"strconv"
	"unicode"
)

// INFO:  https://www.bittorrent.org/beps/bep_0003.html#:~:text=BitTorrent's%20peer%20protocol%20operates%20over%20TCP%20or%20uTP.

// BitTorrent is a peer-to-peer file sharing protocol used for distributing large amounts of data.
// BitTorrent's peer protocol operates over TCP or uTP.
// First step is to build bencoded string

func decodebencode(bencodedString string) (string, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		var firstColonIndex int
		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}

		if firstColonIndex == 0 {
			return "", fmt.Errorf("invalid bencoded string")
		}

		lenghtStr := bencodedString[:firstColonIndex]

		lenght, err := strconv.Atoi(lenghtStr)

		if err != nil {
			return "", err
		}

		return bencodedString[firstColonIndex+1 : firstColonIndex+1+lenght], nil
	} else {
		return "", fmt.Errorf("this is not a right way to represent a string")
	}
}

func main() {

	command := os.Args[1]

	if command == "decode" {

		var bencodedValue string = os.Args[2]

		decode, err := decodebencode(bencodedValue)

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(decode)

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
