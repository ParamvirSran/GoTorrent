package main

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

func ParseTorrentFile() {

	return
}
