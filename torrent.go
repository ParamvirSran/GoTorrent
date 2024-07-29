package main

import (
	"bytes"
	"fmt"
	"os"
)

// Metainfo represents the structure of the .torrent file.
type Metainfo struct {
	Announce string
	Info     Info
}

// Info represents the info dictionary in the .torrent file.
type Info struct {
	Name        string
	PieceLength int
	Pieces      string
	Files       []File
	Length      int
}

// File represents each file in a multi-file torrent.
type File struct {
	Length int
	Path   []string
}

// DecodeTorrentFile takes the bencoded contents of a torrent file and returns the Metainfo struct.
func DecodeTorrentFile(content []byte) (*Metainfo, error) {
	data, err := Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %w", err)
	}

	torrentDict, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format: expected a dictionary but got %T", data)
	}

	return ParseMetainfo(torrentDict)
}

// ParseMetainfo parses the decoded dictionary of the torrent file's contents into the Metainfo struct.
func ParseMetainfo(torrentDict map[string]interface{}) (*Metainfo, error) {
	torrentFile := &Metainfo{}

	// Parsing the announce URL
	announce, ok := torrentDict["announce"].(string)
	if !ok {
		return nil, fmt.Errorf("announce URL missing, not a string, %v", announce)
	}
	torrentFile.Announce = announce

	// Parsing the info dictionary
	infoDict, ok := torrentDict["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info dictionary missing or of incorrect type")
	}
	info, err := ParseInfo(infoDict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse info dictionary: %w", err)
	}
	torrentFile.Info = *info

	return torrentFile, nil
}

// ParseInfo parses the info dictionary into the Info struct.
func ParseInfo(infoDict map[string]interface{}) (*Info, error) {
	info := &Info{}

	// Parsing the name
	name, ok := infoDict["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name field missing, not a string: %v", name)
	}
	info.Name = name

	// Parsing piece length
	switch pl := infoDict["piece length"].(type) {
	case int:
		info.PieceLength = pl
	case int64:
		info.PieceLength = int(pl)
	default:
		return nil, fmt.Errorf("piece length field missing or of incorrect type")
	}

	// Parsing pieces
	pieces, ok := infoDict["pieces"].(string)
	if !ok {
		return nil, fmt.Errorf("pieces field missing, not a string")
	}
	info.Pieces = pieces

	// Parsing length or files
	if length, ok := infoDict["length"].(int); ok {
		info.Length = length
	} else if length, ok := infoDict["length"].(int64); ok {
		info.Length = int(length)
	} else if files, ok := infoDict["files"].([]interface{}); ok {
		parsedFiles, err := ParseFiles(files)
		if err != nil {
			return nil, fmt.Errorf("failed to parse files: %w", err)
		}
		info.Files = parsedFiles
	} else {
		return nil, fmt.Errorf("neither length nor files field found")
	}

	return info, nil
}

// ParseFiles parses the files list into the Files struct.
func ParseFiles(files []interface{}) ([]File, error) {
	var fileList []File
	for _, file := range files {
		fileDict, ok := file.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("file entry is not a valid dictionary, got %T", file)
		}
		fileInfo, err := ParseFile(fileDict)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file: %w", err)
		}
		fileList = append(fileList, fileInfo)
	}
	return fileList, nil
}

// ParseFile parses a single file dictionary into a File struct.
func ParseFile(fileDict map[string]interface{}) (File, error) {
	var fileInfo File

	if length, ok := fileDict["length"].(int); ok {
		fileInfo.Length = length
	} else if length, ok := fileDict["length"].(int64); ok {
		fileInfo.Length = int(length)
	} else {
		return fileInfo, fmt.Errorf("file length missing or of incorrect type")
	}

	if path, ok := fileDict["path"].([]interface{}); ok {
		for _, p := range path {
			if pStr, ok := p.(string); ok {
				fileInfo.Path = append(fileInfo.Path, pStr)
			} else {
				return fileInfo, fmt.Errorf("path element is not a valid type, got %T", p)
			}
		}
	} else {
		return fileInfo, fmt.Errorf("path missing or of incorrect type")
	}

	return fileInfo, nil
}

// ParseTorrentFile parses a torrent file from the provided path and returns a pointer to the Metainfo struct.
func ParseTorrentFile(filePath string) (*Metainfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading .torrent file: %v", err)
	}

	torrentFile, err := DecodeTorrentFile(content)
	if err != nil {
		return nil, fmt.Errorf("error decoding .torrent file into metainfo structure: %v", err)
	}
	return torrentFile, nil
}
