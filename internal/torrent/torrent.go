package torrent

import (
	"bytes"
	"fmt"
	"github.com/ParamvirSran/GoTorrent/internal/decode"
	"os"
)

type Metainfo struct {
	Announce     string
	AnnounceList [][]string
	Info         Info
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

func ParseTorrentFile(torrentPath string) (*Metainfo, error) {
	content, err := os.ReadFile(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("error reading .torrent file: %v", err)
	}

	data, err := decode.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %w", err)
	}

	torrentDict, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format: expected a dictionary but got %T", data)
	}

	return parseMetainfo(torrentDict)
}

func parseMetainfo(torrentDict map[string]interface{}) (*Metainfo, error) {
	torrentFile := &Metainfo{}

	if announce, ok := torrentDict["announce"].(string); ok {
		torrentFile.Announce = announce
	} else {
		return nil, fmt.Errorf("announce URL missing or not a string")
	}

	if announceList, ok := torrentDict["announce-list"].([]interface{}); ok {
		for _, tier := range announceList {
			if urlList, ok := tier.([]interface{}); ok {
				var urls []string
				for _, url := range urlList {
					if urlString, ok := url.(string); ok {
						urls = append(urls, urlString)
					} else {
						return nil, fmt.Errorf("announce-list contains non-string URL: %v", url)
					}
				}
				torrentFile.AnnounceList = append(torrentFile.AnnounceList, urls)
			} else {
				return nil, fmt.Errorf("announce-list tier is not a list of URLs: %T", tier)
			}
		}
	}

	infoDict, ok := torrentDict["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info dictionary missing or of incorrect type")
	}

	info, err := parseInfo(infoDict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse info dictionary: %w", err)
	}
	torrentFile.Info = *info

	return torrentFile, nil
}

func parseInfo(infoDict map[string]interface{}) (*Info, error) {
	info := &Info{}

	name, ok := infoDict["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name field missing or not a string")
	}
	info.Name = name

	switch pl := infoDict["piece length"].(type) {
	case int:
		info.PieceLength = pl
	case int64:
		info.PieceLength = int(pl)
	default:
		return nil, fmt.Errorf("piece length field missing or of incorrect type")
	}

	pieces, ok := infoDict["pieces"].(string)
	if !ok {
		return nil, fmt.Errorf("pieces field missing or not a string")
	}
	info.Pieces = pieces

	if length, ok := infoDict["length"].(int); ok {
		info.Length = length
	} else if length, ok := infoDict["length"].(int64); ok {
		info.Length = int(length)
	} else if files, ok := infoDict["files"].([]interface{}); ok {
		parsedFiles, err := parseFiles(files)
		if err != nil {
			return nil, fmt.Errorf("failed to parse files: %w", err)
		}
		info.Files = parsedFiles
	} else {
		return nil, fmt.Errorf("neither length nor files field found")
	}

	return info, nil
}

func parseFiles(files []interface{}) ([]File, error) {
	var fileList []File
	for _, file := range files {
		fileDict, ok := file.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("file entry is not a valid dictionary, got %T", file)
		}
		fileInfo, err := parseFile(fileDict)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file: %w", err)
		}
		fileList = append(fileList, fileInfo)
	}

	return fileList, nil
}

func parseFile(fileDict map[string]interface{}) (File, error) {
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
