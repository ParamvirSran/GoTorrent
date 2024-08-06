package main

import (
	"fmt"
	"net"
)

func handleCompactPeers(peers string) []string {
	var peerList []string
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4])
		port := uint16(peers[i+4])<<8 | uint16(peers[i+5])
		peerList = append(peerList, fmt.Sprintf("%s:%d", ip, port))
	}
	return peerList
}

func handleDictionaryPeers(peers []interface{}) []string {
	var peerList []string
	for _, peer := range peers {
		peerMap, ok := peer.(map[string]interface{})
		if !ok {
			continue
		}
		ip, ipOk := peerMap["ip"].(string)
		port, portOk := peerMap["port"].(int64)
		if ipOk && portOk {
			peerList = append(peerList, fmt.Sprintf("%s:%d", ip, port))
		}
	}
	return peerList
}
