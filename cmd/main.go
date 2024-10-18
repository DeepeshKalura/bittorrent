package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	bencode "github.com/jackpal/bencode-go"
)

type PeerMessage struct {
	lengthPrefix uint32
	id           uint8
	index        uint32
	begin        uint32
	length       uint32
}

type MetaInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
	PieceLength int    `bencode:"piece length"`
}
type Meta struct {
	Announce string   `bencode:"announce"`
	Info     MetaInfo `bencode:"info"`
}

type TrackerResponse struct {
	Interval int    `json:"interval"`
	Peers    string `json:"peers"`
}

func discoverPeers(file []byte, peerId interface{}) (TrackerResponse, error) {

	var meta Meta

	err := bencode.Unmarshal(bytes.NewReader(file), &meta)

	if err != nil {
		return TrackerResponse{}, err
	}

	h := sha1.New()

	bencode.Marshal(h, meta.Info)

	infoHash := hex.EncodeToString(h.Sum(nil))

	infoHashBytes, _ := hex.DecodeString(infoHash)

	if peerId == nil {
		peerId = randomString(20)
	}

	params := url.Values{}
	params.Add("info_hash", string(infoHashBytes))
	params.Add("peer_id", peerId.(string))
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprint(meta.Info.Length))
	params.Add("compact", "1")

	finalURL := fmt.Sprintf("%s?%s", meta.Announce, params.Encode())

	response, err := http.Get(finalURL)
	if err != nil {
		return TrackerResponse{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return TrackerResponse{}, err
	}

	var trackerResponse TrackerResponse

	err = bencode.Unmarshal(bytes.NewReader(body), &trackerResponse)
	if err != nil {
		return TrackerResponse{}, err
	}

	return trackerResponse, nil
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: program <command> <argument>")
		return
	}
	command := os.Args[1]

	switch command {
	case "decode":
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

	case "info":
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
		fmt.Printf("Piece Length: %d\n", meta.Info.PieceLength)

		fmt.Printf("Pieces hashes:\n")

		for i := 0; i < len(meta.Info.Pieces); i += 20 {
			fmt.Printf("%x\n", meta.Info.Pieces[i:i+20])
		}

	case "peers":
		filename := os.Args[2]

		var result TrackerResponse

		f, err := os.ReadFile(filename)

		if err != nil {
			panic(err)
		}

		result, err = discoverPeers(f, nil)

		if err != nil {
			panic(err)
		}

		peersBinary := []byte(result.Peers)
		for i := 0; i < len(peersBinary); i += 6 {
			if i+6 > len(peersBinary) {
				break
			}
			ip := fmt.Sprintf("%d.%d.%d.%d", peersBinary[i], peersBinary[i+1], peersBinary[i+2], peersBinary[i+3])
			port := binary.BigEndian.Uint16(peersBinary[i+4 : i+6])

			fmt.Printf("%s:%d\n", ip, port)
		}

	case "handshake":

		filename := os.Args[2]
		peerAdress := os.Args[3]

		f, err := os.ReadFile(filename)

		if err != nil {
			panic(err)
		}

		var meta Meta

		err = bencode.Unmarshal(bytes.NewReader(f), &meta)

		if err != nil {
			panic(err)
		}

		conn, err := net.Dial("tcp", peerAdress)

		if err != nil {
			panic(err)
		}

		defer conn.Close()

		h := sha1.New()

		bencode.Marshal(h, meta.Info)

		infoHash := hex.EncodeToString(h.Sum(nil))

		infoHashBytes, _ := hex.DecodeString(infoHash)

		peerId := randomString(20)

		reserveByte := make([]byte, 8)
		pstrlen := byte(19)

		handshake := append([]byte{pstrlen}, []byte("BitTorrent protocol")...)

		handshake = append(handshake, reserveByte...)

		handshake = append(handshake, infoHashBytes...)

		handshake = append(handshake, []byte(peerId)...)

		_, err = conn.Write(handshake)
		if err != nil {
			panic(err)
		}

		// Receive handshake response
		response := make([]byte, 68)
		_, err = conn.Read(response)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Peer ID: %s\n", hex.EncodeToString(response[48:]))

	case "download_piece":
		args := os.Args[2:]

		peerId := randomString(20)

		var torrentFile, outputPath string

		if args[0] == "-o" {
			torrentFile = args[2]
			outputPath = args[1]
		} else {
			torrentFile = args[0]
			outputPath = "."
		}

		f, err := os.ReadFile(torrentFile)

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

		infoHash := hex.EncodeToString(h.Sum(nil))

		infoHashBytes, _ := hex.DecodeString(infoHash)

		peers, err := discoverPeers(f, peerId)

		if err != nil {
			panic(err)
		}

		peerIp := fmt.Sprintf("%d.%d.%d.%d", peers.Peers[0], peers.Peers[1], peers.Peers[2], peers.Peers[3])

		peerPort := int(peers.Peers[4])<<8 | int(peers.Peers[5])
		peerPortStr := fmt.Sprintf("%d", peerPort)

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", peerIp, peerPortStr))

		reserveByte := make([]byte, 8)
		pstrlen := byte(19)

		handshake := append([]byte{pstrlen}, []byte("BitTorrent protocol")...)

		handshake = append(handshake, reserveByte...)

		handshake = append(handshake, infoHashBytes...)

		handshake = append(handshake, []byte(peerId)...)

		_, err = conn.Write(handshake)
		if err != nil {
			panic(err)
		}

		// Receive handshake response
		response := make([]byte, 68)
		_, err = conn.Read(response)
		if err != nil {
			panic(err)
		}

		// fmt.Printf("Peer ID: %s\n", hex.EncodeToString(response[48:]))

		ind, _ := strconv.Atoi(args[3])

		data := downloadFile(conn, meta, ind)

		file, err := os.Create(outputPath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		_, err = file.Write(data)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Piece downloaded to %s.\n", outputPath)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
