package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ParamvirSran/GoTorrent/internal/bencode"
	"github.com/ParamvirSran/GoTorrent/internal/peers"
)

// Constants for repeated values
const (
	PeerIDPrefix = "-GO-"
	// CompactParam = "compact"
)

// TrackerConfig defines configurable parameters for tracker communication
type TrackerConfig struct {
	Timeout time.Duration
}

// ContactTrackers tries to contact multiple trackers and gather peers
func ContactTrackers(trackers []string, infoHash, peerID, event string, uploaded, downloaded, left, port int) ([]string, []string, error) {
	var peer_address_list []string
	var peerID_list []string

	for _, trackerURL := range trackers {
		tracker_peerID_list, tracker_peerIP_list, err := extractPeersFromTracker(trackerURL, infoHash, peerID, event, uploaded, downloaded, left, port)
		if err != nil {
			log.Printf("Error contacting tracker %s: %v", trackerURL, err)
			continue
		}

		if len(tracker_peerIP_list) > 0 && len(tracker_peerID_list) > 0 && len(tracker_peerID_list) == len(tracker_peerIP_list) {
			peer_address_list = append(peer_address_list, tracker_peerIP_list...)
			peerID_list = append(peerID_list, tracker_peerID_list...)
			event = "" // After the first successful announce, the event should be empty
		}
		if len(tracker_peerIP_list) > 0 && len(tracker_peerID_list) == 0 {
			peer_address_list = append(peer_address_list, tracker_peerIP_list...)

			unknown_peerID_list := make([]string, len(tracker_peerIP_list))
			for i := 0; i < len(tracker_peerIP_list); i++ {
				unknown_peerID_list[i] = ""
			}
			peerID_list = append(peerID_list, unknown_peerID_list...)
		}
	}
	if len(peer_address_list) == 0 {
		return nil, nil, fmt.Errorf("no valid peers found from any tracker")
	}
	return peerID_list, peer_address_list, nil
}

// GatherTrackers extracts and returns a list of tracker URLs from the torrent file
func GatherTrackers(torrentFile *Torrent) []string {
	trackers := []string{torrentFile.Announce}
	for _, tier := range torrentFile.AnnounceList {
		for _, t := range tier {
			tURL, err := url.Parse(t)
			if err == nil && tURL.Scheme != "udp" {
				trackers = append(trackers, tURL.String())
			}
		}
	}
	return trackers[1:]
}

// GeneratePeerID creates a random peer ID with a fixed prefix "-GO-" and a random 8-byte part.
// The resulting ID will be 20 characters long, including the prefix.
func GeneratePeerID() (string, error) {
	random := make([]byte, 8) // 8 random bytes (16 hex characters + 4 for the prefix)
	_, err := rand.Read(random)
	if err != nil {
		return "", fmt.Errorf("error generating random peer ID: %w", err)
	}

	// Combine the prefix with the random part
	peerID := PeerIDPrefix + hex.EncodeToString(random) // This should result in exactly 20 characters

	if len(peerID) != 20 {
		return "", fmt.Errorf("generated peer ID has invalid length: %d", len(peerID))
	}

	// Escape the peer ID (URL-encode)
	escapedPeerID := url.QueryEscape(peerID)

	return escapedPeerID, nil
}

// GetInfoHash calculates the SHA-1 hash of the bencoded "info" dictionary
func GetInfoHash(info InfoDictionary) ([]byte, error) {
	infoMap := make(map[string]interface{})
	infoMap["name"] = info.Name
	infoMap["piece length"] = info.PieceLength
	infoMap["pieces"] = string(info.Pieces)

	if info.Length > 0 {
		// Single-file case
		infoMap["length"] = info.Length
	} else {
		// Multi-file case
		var files []interface{}
		for _, file := range info.Files {
			pathInterfaceList := make([]interface{}, len(file.Path))
			for i, p := range file.Path {
				pathInterfaceList[i] = p
			}
			fileMap := map[string]interface{}{
				"length": file.Length,
				"path":   pathInterfaceList,
			}
			files = append(files, fileMap)
		}
		infoMap["files"] = files
	}

	// bencode the info dict
	encodedInfo, err := bencode.Encode(infoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to encode info dictionary: %w", err)
	}

	// compute the SHA-1 hash of the bencoded dict
	hash := sha1.Sum(encodedInfo)
	return hash[:], nil // converting the [20]byte hash to []byte
}

// newTrackerClient creates a new HTTP client with a customizable timeout
func newTrackerClient(config *TrackerConfig) *http.Client {
	if config == nil {
		config = &TrackerConfig{Timeout: 10 * time.Second} // default timeout
	}
	return &http.Client{
		Timeout: config.Timeout,
	}
}

// buildAnnounceURL creates the announcement URL for sending to trackers
func buildAnnounceURL(baseURL, infoHash, peerID, event string, uploaded, downloaded, left, port int) (string, error) {
	// validate mandatory parameters
	if baseURL == "" || infoHash == "" || peerID == "" {
		return "", fmt.Errorf("missing required parameters: baseURL, infoHash, or peerID")
	}

	// parse the base URL
	trackerURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid tracker base URL: %w", err)
	}

	// construct query parameters
	params := url.Values{}
	addQueryParam(params, "info_hash", infoHash)
	addQueryParam(params, "peer_id", peerID)
	addQueryParam(params, "port", strconv.Itoa(port))
	addQueryParam(params, "uploaded", strconv.Itoa(uploaded))
	addQueryParam(params, "downloaded", strconv.Itoa(downloaded))
	addQueryParam(params, "left", strconv.Itoa(left))
	addQueryParam(params, "compact", strconv.Itoa(0))
	// params.Add(CompactParam, "0") // request compact peer list

	if event != "" {
		addQueryParam(params, "event", event)
	}

	// attach query parameters to the URL
	trackerURL.RawQuery = params.Encode()

	// return the full announce URL
	return trackerURL.String(), nil
}

// addQueryParam adds a query parameter to the URL if the value is not empty
func addQueryParam(params url.Values, key, value string) {
	if value != "" {
		params.Add(key, value)
	}
}

// sendGetRequest sends a GET request to the tracker
func sendGetRequest(url string, client *http.Client) ([]byte, error) {
	// send the GET request
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending GET request: %w", err)
	}
	defer response.Body.Close()

	// check if the response status code is 200 OK
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", response.Status)
	}

	// read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil // return response body
}

// parseTrackerResponse decodes the response from the tracker
func parseTrackerResponse(response []byte) (map[string]interface{}, error) {
	decoded, err := bencode.Decode(bytes.NewReader(response))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	trackerResponse, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tracker response format: expected dictionary but got %T", decoded)
	}
	if failReason, ok := trackerResponse["failure reason"].(string); ok {
		return nil, fmt.Errorf("tracker failure reason %s", failReason)
	}
	return trackerResponse, nil
}

// extractPeersFromTracker sends a request to the tracker and extracts peers
func extractPeersFromTracker(trackerURL, infoHash, peerID, event string, uploaded, downloaded, left, port int) ([]string, []string, error) {
	requestURL, err := buildAnnounceURL(trackerURL, infoHash, peerID, event, uploaded, downloaded, left, port)
	if err != nil {
		return nil, nil, fmt.Errorf("error building announce URL: %w", err)
	}

	client := newTrackerClient(nil) // Default timeout
	resp, err := sendGetRequest(requestURL, client)
	if err != nil {
		return nil, nil, fmt.Errorf("error sending GET request: %w", err)
	}

	trackerResp, err := parseTrackerResponse(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing tracker response: %w", err)
	}

	peer_id_list, peerList, err := peers.ExtractPeers(trackerResp)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting peers: %w", err)
	}

	return peer_id_list, peerList, nil
}
