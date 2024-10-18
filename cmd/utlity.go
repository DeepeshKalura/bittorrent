package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
)

func downloadFile(conn net.Conn, torrent Meta, index int) []byte {
	defer conn.Close()

	// wait for the bitfield message (id = 5)
	buf := make([]byte, 4)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	peerMessage := PeerMessage{}
	peerMessage.lengthPrefix = binary.BigEndian.Uint32(buf)
	payloadBuf := make([]byte, peerMessage.lengthPrefix)
	_, err = conn.Read(payloadBuf)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	peerMessage.id = payloadBuf[0]

	fmt.Printf("Received message: %v\n", peerMessage)
	if peerMessage.id != 5 {
		fmt.Println("Expected bitfield message")
		return nil
	}

	// send interested message (id = 2)
	_, err = conn.Write([]byte{0, 0, 0, 1, 2})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// wait for unchoke message (id = 1)
	buf = make([]byte, 4)
	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	peerMessage = PeerMessage{}
	peerMessage.lengthPrefix = binary.BigEndian.Uint32(buf)
	payloadBuf = make([]byte, peerMessage.lengthPrefix)
	_, err = conn.Read(payloadBuf)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	peerMessage.id = payloadBuf[0]

	fmt.Printf("Received message: %v\n", peerMessage)
	if peerMessage.id != 1 {
		fmt.Println(buf)
		fmt.Println("Expected unchoke message")
		return nil
	}

	// send request message (id = 6) for each block
	// Break the piece into blocks of 16 kiB (16 * 1024 bytes) and send a request message for each block
	pieceSize := torrent.Info.PieceLength
	pieceCnt := int(math.Ceil(float64(torrent.Info.Length) / float64(pieceSize)))
	if index == pieceCnt-1 {
		pieceSize = torrent.Info.Length % torrent.Info.PieceLength
	}
	blockSize := 16 * 1024
	blockCnt := int(math.Ceil(float64(pieceSize) / float64(blockSize)))
	fmt.Printf("File Length: %d, Piece Length: %d, Piece Count: %d, Block Size: %d, Block Count: %d\n", torrent.Info.Length, torrent.Info.PieceLength, pieceCnt, blockSize, blockCnt)
	var data []byte
	for i := 0; i < blockCnt; i++ {
		blockLength := blockSize
		if i == blockCnt-1 {
			blockLength = pieceSize - ((blockCnt - 1) * int(blockSize))
		}
		peerMessage := PeerMessage{
			lengthPrefix: 13,
			id:           6,
			index:        uint32(index),
			begin:        uint32(i * int(blockSize)),
			length:       uint32(blockLength),
		}

		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, peerMessage)
		_, err = conn.Write(buf.Bytes())
		if err != nil {
			fmt.Println(err)
			return nil
		}
		fmt.Println("Sent request message", peerMessage)

		// wait for piece message (id = 7)
		resBuf := make([]byte, 4)
		_, err = conn.Read(resBuf)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		peerMessage = PeerMessage{}
		peerMessage.lengthPrefix = binary.BigEndian.Uint32(resBuf)
		payloadBuf := make([]byte, peerMessage.lengthPrefix)
		_, err = io.ReadFull(conn, payloadBuf)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		peerMessage.id = payloadBuf[0]
		fmt.Printf("Received message: %v\n", peerMessage)

		data = append(data, payloadBuf[9:]...)
	}

	return data
}
