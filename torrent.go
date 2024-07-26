package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
)

type Metainfo struct {
	Announce string
	Info     Info
}

type Info struct {
	Name        string
	PieceLength int
	Pieces      string
	Files       []File
	Length      int
}

type File struct {
	Length int
	Path   []string
}

// DecodeTorrentFile decodes the bencoded torrent file content
func DecodeTorrentFile(content []byte) (*Metainfo, error) {
	data, err := Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %w", err)
	}

	torrentDict, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format, could not map correctly")
	}

	torrentFile := &Metainfo{}

	if announce, ok := torrentDict["announce"].(string); ok {
		torrentFile.Announce = announce
	}

	if infoDict, ok := torrentDict["info"].(map[string]interface{}); ok {

		info := Info{}

		if name, ok := infoDict["name"].(string); ok {
			info.Name = name
		}
		if pieceLength, ok := infoDict["piece length"].(int); ok {
			info.PieceLength = pieceLength
		} else if pieceLength, ok := infoDict["piece length"].(int64); ok {
			info.PieceLength = int(pieceLength)
		}
		if pieces, ok := infoDict["pieces"].([]byte); ok {
			info.Pieces = hex.EncodeToString(pieces)
		}
		if length, ok := infoDict["length"].(int); ok {
			info.Length = length
		} else if length, ok := infoDict["length"].(int64); ok {
			info.Length = int(length)
		}
		if files, ok := infoDict["files"].([]interface{}); ok {
			for _, file := range files {
				if fileDict, ok := file.(map[string]interface{}); ok {

					fileInfo := File{}

					if length, ok := fileDict["length"].(int); ok {
						fileInfo.Length = length
					} else if length, ok := fileDict["length"].(int64); ok {
						fileInfo.Length = int(length)
					}
					if path, ok := fileDict["path"].([]interface{}); ok {
						for _, p := range path {
							if pStr, ok := p.(string); ok {
								fileInfo.Path = append(fileInfo.Path, pStr)
							}
						}
					}
					info.Files = append(info.Files, fileInfo)
				}
			}
		}
		torrentFile.Info = info
	}

	return torrentFile, nil
}

// ParseTorrentFile parses a torrent file and prints the decoded information
func ParseTorrentFile(filePath string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("error reading .torrent file: %v\n", err)
		return
	}

	torrentFile, err := DecodeTorrentFile(content)
	if err != nil {
		fmt.Printf("error decoding .torrent file into metainfo structure: %v\n", err)
		return
	}

	fmt.Printf("Torrent File: %+v\n", torrentFile)
}

func main() {
	ParseTorrentFile("test.torrent")
}
