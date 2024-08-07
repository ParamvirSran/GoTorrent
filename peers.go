package main

import (
	"fmt"
	"net"
)

func ParseCompactPeers(peers []byte) ([]string, error) {
	if len(peers)%6 != 0 {
		return nil, fmt.Errorf("invalid compact peers length")
	}

	var peerList []string
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4]).String()
		port := int(peers[i+4])<<8 + int(peers[i+5])
		peer := fmt.Sprintf("%s:%d", ip, port)
		peerList = append(peerList, peer)
	}

	return peerList, nil
}

func ParseDictionaryPeers(peers []interface{}) ([]string, error) {
	var peerList []string
	for _, peer := range peers {
		peerMap, ok := peer.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid peer format")
		}
		ip, ipOk := peerMap["ip"].(string)
		port, portOk := peerMap["port"].(int)
		if ipOk && portOk {
			peerList = append(peerList, fmt.Sprintf("%s:%d", ip, port))
		} else {
			return nil, fmt.Errorf("invalid peer format: missing ip or port")
		}
	}
	return peerList, nil
}
