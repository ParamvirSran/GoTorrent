package main

import (
	"crypto/sha1"
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
