package pbscommon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cornelk/hashmap"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/http2"
)

type IndexCreateResp struct {
	WriterID int `json:"data"`
}

type IndexPutReq struct {
	DigestList []string `json:"digest-list"`
	OffsetList []uint64 `json:"offset-list"`
	WriterID   uint64   `json:"wid"`
}

type IndexCloseReq struct {
	ChunkCount uint64 `json:"chunk-count"`
	CheckSum   string `json:"csum"`
	Size       uint64 `json:"size"`
	WriterID   uint64 `json:"wid"`
}

type File struct {
	CryptMode string `json:"crypt-mode"`
	Csum      string `json:"csum"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
}

type ChunkUploadStats struct {
	CompressedSize int64 `json:"compressed_size"`
	Count          int   `json:"count"`
	Duplicates     int   `json:"duplicates"`
	Size           int64 `json:"size"`
}

type FixedIndexCreateReq struct {
	ArchiveName string `json:"archive-name"`
	Size        int64  `json:"size"`
}

type Unprotected struct {
	ChunkUploadStats ChunkUploadStats `json:"chunk_upload_stats"`
}

type BackupManifest struct {
	BackupID    string      `json:"backup-id"`
	BackupTime  int64       `json:"backup-time"`
	BackupType  string      `json:"backup-type"`
	Files       []File      `json:"files"`
	Signature   interface{} `json:"signature"`
	Unprotected Unprotected `json:"unprotected"`
}

type AuthErr struct {
	StatusCode   string
	ResponseBody string
}

func (e *AuthErr) Error() string {
	return fmt.Sprintf("PBS authentication failed: HTTP %s - %s", e.StatusCode, e.ResponseBody)
}

type PBSClient struct {
	BaseURL         string
	CertFingerPrint string
	APIToken        string
	Secret          string
	AuthID          string

	Datastore string
	Namespace string
	Manifest  BackupManifest

	Insecure bool

	Client    http.Client
	TLSConfig tls.Config

	WritersManifest map[uint64]int
	SkippedFiles    []string // Track files/dirs skipped during backup
}

const PBS_FIXED_CHUNK_SIZE = 4 * 1024 * 1024

var blobCompressedMagic = []byte{49, 185, 88, 66, 111, 182, 163, 127}
var blobUncompressedMagic = []byte{66, 171, 56, 7, 190, 131, 112, 161}

type SnapshotsResp struct {
	Data []BackupManifest `json:"data"`
}

func (pbs *PBSClient) ListSnapshots() ([]BackupManifest, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	if pbs.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client.Transport = tr
	}

	ret := make([]BackupManifest, 0)
	var r SnapshotsResp
	params := url.Values{}
	params.Add("ns", pbs.Namespace)
	fullURL := fmt.Sprintf("%s/api2/json/admin/datastore/%s/snapshots?%s", pbs.BaseURL, pbs.Datastore, params.Encode())

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return ret, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	resp, err := client.Do(req)
	if err != nil {
		return ret, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return ret, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return ret, err
	}
	return r.Data, nil

}

func (pbs *PBSClient) CreateFixedIndex(fic FixedIndexCreateReq) (uint64, error) {
	jd, err := json.Marshal(fic)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest("POST", pbs.BaseURL+"/fixed_index", bytes.NewBuffer(jd))
	if err != nil {
		return 0, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return 0, err
	}

	if resp2.StatusCode != http.StatusOK {
		resp1, _ := io.ReadAll(resp2.Body)
		fmt.Println("Error making request:", string(resp1), string(resp2.Proto))
		return 0, fmt.Errorf("Error making request:", string(resp1), string(resp2.Proto))
	}

	resp1, err := io.ReadAll(resp2.Body)
	if err != nil {
		return 0, err
	}
	var R IndexCreateResp
	err = json.Unmarshal(resp1, &R)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return 0, err
	}
	fmt.Println("Writer id: ", R.WriterID)
	defer resp2.Body.Close()
	f := File{
		CryptMode: "none",
		Csum:      "",
		Filename:  fic.ArchiveName,
		Size:      0,
	}
	pbs.Manifest.Files = append(pbs.Manifest.Files, f)
	pbs.WritersManifest[uint64(R.WriterID)] = len(pbs.Manifest.Files) - 1
	return uint64(R.WriterID), nil

}

func (pbs *PBSClient) AssignFixedChunks(writerid uint64, digests []string, offsets []uint64) error {
	indexput := &IndexPutReq{
		WriterID:   writerid,
		DigestList: digests,
		OffsetList: offsets,
	}

	jsondata, err := json.Marshal(indexput)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", pbs.BaseURL+"/fixed_index", bytes.NewBuffer(jsondata))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}
	defer resp2.Body.Close()
	return nil
}

func (pbs *PBSClient) CloseFixedIndex(writerid uint64, checksum string, totalsize uint64, chunkcount uint64) error {
	finishreq := &IndexCloseReq{
		WriterID:   writerid,
		CheckSum:   checksum,
		Size:       totalsize,
		ChunkCount: chunkcount,
	}
	jsonpayload, err := json.Marshal(finishreq)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", pbs.BaseURL+"/fixed_close", bytes.NewBuffer(jsonpayload))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}

	f := &pbs.Manifest.Files[pbs.WritersManifest[writerid]]

	f.Csum = checksum
	f.Size = int64(totalsize)

	defer resp2.Body.Close()
	return nil
}

func (pbs *PBSClient) CreateDynamicIndex(name string) (uint64, error) {
	fmt.Printf("=== CreateDynamicIndex START ===\n")
	fmt.Printf("Archive name: %s\n", name)
	fmt.Printf("BaseURL: %s\n", pbs.BaseURL)

	req, err := http.NewRequest("POST", pbs.BaseURL+"/dynamic_index", bytes.NewBuffer([]byte(fmt.Sprintf("{\"archive-name\": \"%s\"}", name))))
	if err != nil {
		fmt.Printf("ERROR: Failed to create request: %v\n", err)
		return 0, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	fmt.Printf("Sending POST request to: %s\n", req.URL.String())
	fmt.Printf("Headers: %+v\n", req.Header)

	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: HTTP request failed: %v\n", err)
		fmt.Printf("ERROR: Error type: %T\n", err)
		return 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp2.Body.Close()

	fmt.Printf("Response status: %d %s\n", resp2.StatusCode, resp2.Status)
	fmt.Printf("Response proto: %s\n", resp2.Proto)

	if resp2.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp2.Body)
		fmt.Printf("ERROR: PBS returned non-200 status\n")
		fmt.Printf("Status: %d\n", resp2.StatusCode)
		fmt.Printf("Body: %s\n", string(bodyBytes))
		return 0, fmt.Errorf("PBS returned HTTP %d: %s", resp2.StatusCode, string(bodyBytes))
	}

	resp1, err := io.ReadAll(resp2.Body)
	if err != nil {
		return 0, err
	}
	var R IndexCreateResp
	err = json.Unmarshal(resp1, &R)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return 0, err
	}
	fmt.Println("Writer id: ", R.WriterID)
	defer resp2.Body.Close()
	f := File{
		CryptMode: "none",
		Csum:      "",
		Filename:  name,
		Size:      0,
	}
	pbs.Manifest.Files = append(pbs.Manifest.Files, f)
	pbs.WritersManifest[uint64(R.WriterID)] = len(pbs.Manifest.Files) - 1
	return uint64(R.WriterID), nil
}

func (pbs *PBSClient) UploadDynamicUncompressedChunk(writerid uint64, digest string, chunkdata []byte) error {
	return pbs.UploadChunk(writerid, digest, chunkdata, true, false)
}
func (pbs *PBSClient) UploadFixedUncompressedChunk(writerid uint64, digest string, chunkdata []byte) error {
	return pbs.UploadChunk(writerid, digest, chunkdata, false, false)
}
func (pbs *PBSClient) UploadDynamicCompressedChunk(writerid uint64, digest string, chunkdata []byte) error {
	return pbs.UploadChunk(writerid, digest, chunkdata, true, true)
}
func (pbs *PBSClient) UploadFixedCompressedChunk(writerid uint64, digest string, chunkdata []byte) error {
	return pbs.UploadChunk(writerid, digest, chunkdata, false, true)
}

func (pbs *PBSClient) UploadChunk(writerid uint64, digest string, chunkdata []byte, dynamic bool, compressed bool) error {
	outBuffer := make([]byte, 0)
	if compressed {
		outBuffer = append(outBuffer, blobCompressedMagic...)
		compressedData := make([]byte, 0)

		//opt := zstd.WithEncoderLevel(zstd.SpeedFastest)
		w, _ := zstd.NewWriter(nil)
		compressedData = w.EncodeAll(chunkdata, compressedData)
		checksum := crc32.Checksum(compressedData, crc32.IEEETable)
		//binary.Write(outBuffer, binary.LittleEndian, checksum)
		outBuffer = binary.LittleEndian.AppendUint32(outBuffer, checksum)

		//fmt.Printf("Appended checksum %08x , len: %d\n", checksum, len(outBuffer))

		outBuffer = append(outBuffer, compressedData...)

		if len(compressedData) > len(chunkdata) {
			return pbs.UploadChunk(writerid, digest, chunkdata, dynamic, false)
		}
	} else {
		outBuffer = append(outBuffer, blobUncompressedMagic...)
		checksum := crc32.Checksum(chunkdata, crc32.IEEETable)
		outBuffer = binary.LittleEndian.AppendUint32(outBuffer, checksum)
		outBuffer = append(outBuffer, chunkdata...)
	}

	//fmt.Printf("Compressed: %d , Orig: %d\n", len(compressedData), len(chunkdata))

	q := &url.Values{}
	q.Add("digest", digest)
	q.Add("encoded-size", fmt.Sprintf("%d", len(outBuffer)))
	q.Add("size", fmt.Sprintf("%d", len(chunkdata)))
	q.Add("wid", fmt.Sprintf("%d", writerid))
	suburl := "/dynamic_chunk?"
	if !dynamic {
		suburl = "/fixed_chunk?"
	}
	req, err := http.NewRequest("POST", pbs.BaseURL+suburl+q.Encode(), bytes.NewBuffer(outBuffer))
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}

	if resp2.StatusCode != http.StatusOK {
		resp1, _ := io.ReadAll(resp2.Body)
		fmt.Println("Error making request:", string(resp1), string(resp2.Proto))
		return fmt.Errorf("Error making request: %s %s", string(resp1), string(resp2.Proto))
	}

	return nil
}

func (pbs *PBSClient) AssignDynamicChunks(writerid uint64, digests []string, offsets []uint64) error {
	indexput := &IndexPutReq{
		WriterID:   writerid,
		DigestList: digests,
		OffsetList: offsets,
	}

	jsondata, err := json.Marshal(indexput)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", pbs.BaseURL+"/dynamic_index", bytes.NewBuffer(jsondata))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}
	defer resp2.Body.Close()
	return nil
}

func (pbs *PBSClient) CloseDynamicIndex(writerid uint64, checksum string, totalsize uint64, chunkcount uint64) error {
	finishreq := &IndexCloseReq{
		WriterID:   writerid,
		CheckSum:   checksum,
		Size:       totalsize,
		ChunkCount: chunkcount,
	}
	jsonpayload, err := json.Marshal(finishreq)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", pbs.BaseURL+"/dynamic_close", bytes.NewBuffer(jsonpayload))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}

	f := &pbs.Manifest.Files[pbs.WritersManifest[writerid]]

	f.Csum = checksum
	f.Size = int64(totalsize)

	defer resp2.Body.Close()
	return nil
}

func (pbs *PBSClient) UploadBlob(name string, data []byte) error {
	out := make([]byte, 0)
	out = append(out, blobUncompressedMagic...)

	checksum := crc32.ChecksumIEEE(data)
	out = binary.LittleEndian.AppendUint32(out, checksum)
	out = append(out, data...)

	q := &url.Values{}
	q.Add("encoded-size", fmt.Sprintf("%d", len(out)))
	q.Add("file-name", name)

	req, _ := http.NewRequest("POST", pbs.BaseURL+"/blob?"+q.Encode(), bytes.NewBuffer(out))

	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}

	if resp2.StatusCode != http.StatusOK {
		resp1, err := io.ReadAll(resp2.Body)
		fmt.Println("Error making request:", string(resp1), string(resp2.Proto))
		return err
	}

	pbs.Manifest.Files = append(pbs.Manifest.Files, File{
		CryptMode: "none",
		Csum:      "",
		Filename:  name,
		Size:      int64(len(data)),
	})

	return nil
}

func (pbs *PBSClient) UploadManifest() error {
	manifestBin, err := json.Marshal(pbs.Manifest)
	if err != nil {
		return err
	}
	return pbs.UploadBlob("index.json.blob", manifestBin)
}

func (pbs *PBSClient) Finish() error {
	req, err := http.NewRequest("POST", pbs.BaseURL+"/finish", nil)
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	if err != nil {
		return err
	}
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		if err != nil {
			return err
		}
	}
	defer resp2.Body.Close()
	return nil
}

// TestConnection performs a real HTTP request to verify PBS connectivity
// Returns error if hostname unreachable, credentials invalid, or datastore inaccessible
func (pbs *PBSClient) TestConnection() error {
	// Create TLS config for the test
	tlsConfig := &tls.Config{
		InsecureSkipVerify: pbs.Insecure || pbs.CertFingerPrint == "",
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Test endpoint: GET datastore status (requires auth + datastore access)
	testURL := fmt.Sprintf("%s/api2/json/admin/datastore/%s/status", pbs.BaseURL, pbs.Datastore)

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add PBS authentication header
	req.Header.Set("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed: invalid credentials")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("access denied: check datastore permissions")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server error: HTTP %d", resp.StatusCode)
	}

	return nil
}

func (pbs *PBSClient) Connect(reader bool, backuptype string) {
	pbs.WritersManifest = make(map[uint64]int)
	pbs.SkippedFiles = []string{} // Reset skipped files for new backup
	pbs.TLSConfig = tls.Config{
		InsecureSkipVerify: pbs.Insecure,
	}
	if pbs.Insecure {
		pbs.TLSConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// Extract the peer certificate
			if len(rawCerts) == 0 {
				return fmt.Errorf("no certificates presented by the peer")
			}
			peerCert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse certificate: %v", err)
			}

			// Calculate the SHA-256 fingerprint of the certificate
			expectedFingerprint := strings.ReplaceAll(pbs.CertFingerPrint, ":", "")
			calculatedFingerprint := sha256.Sum256(peerCert.Raw)

			// Compare the calculated fingerprint with the expected one
			if hex.EncodeToString(calculatedFingerprint[:]) != expectedFingerprint && !pbs.Insecure {
				return fmt.Errorf("certificate fingerprint does not match (%s,%s)", expectedFingerprint, hex.EncodeToString(calculatedFingerprint[:]))
			}

			// If the fingerprint matches, the certificate is considered valid
			return nil
		}
	}
	if !reader {
		pbs.Manifest.BackupTime = time.Now().Unix()
	}
	pbs.Manifest.BackupType = backuptype
	if pbs.Manifest.BackupID == "" {
		hostname, _ := os.Hostname()
		pbs.Manifest.BackupID = hostname
	}

	// Close any existing HTTP/2 connections from previous backups (successful or failed)
	// This prevents reusing stale/broken connections that might cause 400 errors
	if pbs.Client.Transport != nil {
		// Close all idle connections first
		pbs.Client.CloseIdleConnections()

		// If it's an http2.Transport, force close it
		if h2Transport, ok := pbs.Client.Transport.(*http2.Transport); ok {
			h2Transport.CloseIdleConnections()
		}

		// Nil out the transport to ensure it's garbage collected
		pbs.Client.Transport = nil
	}

	// Create completely new client with fresh HTTP/2 transport
	pbs.Client = http.Client{
		Transport: &http2.Transport{

			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {

				//This is one of the trickiest parts, GO http2 library does not support starting with http1 and upgrading to 2 after
				//So to achieve that the function to create SSL socket has been hijacked here
				//Here an http 1.1 request to authenticate, start the backup and require upgrade to HTTP2 is done then the socket is passed to
				// http2.Transport handler
				conn, err := tls.Dial(network, addr, &pbs.TLSConfig)
				if err != nil {
					return nil, err
				}
				q := &url.Values{}
				q.Add("backup-time", fmt.Sprintf("%d", pbs.Manifest.BackupTime))
				q.Add("backup-type", pbs.Manifest.BackupType)
				q.Add("store", pbs.Datastore)
				if pbs.Namespace != "" {
					q.Add("ns", pbs.Namespace)
				}

				q.Add("backup-id", pbs.Manifest.BackupID)
				fmt.Println(q.Encode())
				//q.Add("debug", "1")

				// Build and log the full HTTP request
				var requestLines []string
				if !reader {
					requestLines = append(requestLines, "GET /api2/json/backup?"+q.Encode()+" HTTP/1.1")
				} else {
					requestLines = append(requestLines, "GET /api2/json/reader?"+q.Encode()+" HTTP/1.1")
				}
				requestLines = append(requestLines, "Host: "+addr)
				requestLines = append(requestLines, "Authorization: "+fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
				if !reader {
					requestLines = append(requestLines, "Upgrade: proxmox-backup-protocol-v1")
				} else {
					requestLines = append(requestLines, "Upgrade: proxmox-backup-reader-protocol-v1")
				}
				requestLines = append(requestLines, "Connection: Upgrade")

				fullRequest := strings.Join(requestLines, "\r\n") + "\r\n\r\n"
				fmt.Printf("=== SENDING HTTP REQUEST TO PBS ===\n%s=== END REQUEST ===\n", fullRequest)

				// Send the request
				conn.Write([]byte(fullRequest))
				fmt.Print("Reading response to upgrade...\n")
				buf := make([]byte, 0)
				for !strings.HasSuffix(string(buf), "\r\n\r\n") && !strings.HasSuffix(string(buf), "\n\n") {
					//fmt.Println(buf)
					b2 := make([]byte, 1)
					nbytes, err := conn.Read(b2)
					if err != nil || nbytes == 0 {
						fmt.Println("Connection unexpectedly closed")
						return nil, err
					}
					buf = append(buf, b2[:nbytes]...)

					//fmt.Println(string(b2))
				}
				fmt.Printf("=== RECEIVED HTTP RESPONSE FROM PBS ===\n%s\n=== END RESPONSE ===\n", string(buf))

				lines := strings.Split(string(buf), "\n")

				if len(lines) > 0 {
					toks := strings.Split(lines[0], " ")
					if len(toks) > 1 && toks[1] != "101" {
						statusCode := strings.Join(toks[1:], " ")
						fmt.Printf("ERROR: PBS rejected upgrade with status: %s\n", statusCode)
						fmt.Printf("Full response body:\n%s\n", string(buf))
						return nil, &AuthErr{
							StatusCode:   statusCode,
							ResponseBody: string(buf),
						}
					}
				}

				fmt.Printf("Upgraderesp: %s\n", string(buf))
				fmt.Println("Successfully upgraded to HTTP/2.")
				return conn, nil
			},
		},
	}

}

type FIDXHeader struct {
	Magic        [8]byte
	UUID         [16]byte
	CreationTime uint64
	IndexCsum    [32]byte
	Size         uint64
	ChunkSize    uint64
	Padding      [4016]byte
}

func (pbs *PBSClient) DownloadPreviousToBytes(archivename string) ([]byte, error) { //In the future also download to tmp if index is extremely big...
	q := &url.Values{}

	q.Add("archive-name", archivename)

	req, err := http.NewRequest("GET", pbs.BaseURL+"/previous?"+q.Encode(), nil)
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	if err != nil {
		return nil, err
	}
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp2.Body.Close()

	ret, err := io.ReadAll(resp2.Body)

	if err != nil {
		return nil, err
	}

	return ret, nil

}

func (pbs *PBSClient) DownloadToBytes(archivename string) ([]byte, error) { //In the future also download to tmp if index is extremely big...
	q := &url.Values{}

	q.Add("file-name", archivename)

	req, err := http.NewRequest("GET", pbs.BaseURL+"/download?"+q.Encode(), nil)
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	if err != nil {
		return nil, err
	}
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp2.Body.Close()

	ret, err := io.ReadAll(resp2.Body)

	if err != nil {
		return nil, err
	}

	return ret, nil

}

func (pbs *PBSClient) GetKnownSha265FromFIDX(archivename string) (*hashmap.Map[string, bool], error) {
	data, err := pbs.DownloadPreviousToBytes(archivename)
	if err != nil {
		return nil, err
	}
	rdr := bytes.NewReader(data)
	var hdr FIDXHeader
	err = binary.Read(rdr, binary.LittleEndian, &hdr)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(hdr.Magic[:], []byte{47, 127, 65, 237, 145, 253, 15, 205}) {
		return nil, fmt.Errorf("FIDX: Invalid magic %+v", hdr.Magic)
	}
	ret := hashmap.New[string, bool]()
	for i := uint64(0); i < hdr.Size/hdr.ChunkSize; i++ {
		H := make([]byte, 32)
		nbytes, err := rdr.Read(H)
		if err != nil {
			return nil, err
		}
		if nbytes != len(H) {
			return nil, fmt.Errorf("FIDX: Short read")
		}
		ret.Insert(hex.EncodeToString(H), true)
	}
	return ret, nil

}

func (pbs *PBSClient) GetChunkData(digest string) ([]byte, error) {
	q := &url.Values{}

	q.Add("digest", digest)

	req, err := http.NewRequest("GET", pbs.BaseURL+"/chunk?"+q.Encode(), nil)
	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))
	if err != nil {
		return nil, err
	}
	resp2, err := pbs.Client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp2.Body.Close()

	ret, err := io.ReadAll(resp2.Body)

	if err != nil {
		return nil, err
	}

	if slices.Equal(ret[:8], blobUncompressedMagic) {
		return ret[12:], nil
	} else if slices.Equal(ret[:8], blobCompressedMagic) {
		rd1 := bytes.NewReader(ret[12:])
		dec, err := zstd.NewReader(rd1)

		if err != nil {
			return nil, err
		}
		defer dec.Close()
		ret2 := make([]byte, 0)
		ret2, err = dec.DecodeAll(ret[12:], ret2)
		if err != nil {
			return nil, err
		}
		return ret2, nil
	} else {
		return nil, fmt.Errorf("Encrypted chunks not supported!")
	}

}

// KeepAlive sends a lightweight request to PBS to keep the session active
// This prevents the dynamic writer from being expired during long backups
func (pbs *PBSClient) KeepAlive() error {
	// Use a simple API call that doesn't modify anything
	// GET /api2/json/version is perfect - it's lightweight and always available
	req, err := http.NewRequest("GET", pbs.BaseURL+"/api2/json/version", nil)
	if err != nil {
		return fmt.Errorf("failed to create keep-alive request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", pbs.AuthID, pbs.Secret))

	resp, err := pbs.Client.Do(req)
	if err != nil {
		return fmt.Errorf("keep-alive request failed: %w", err)
	}
	defer resp.Body.Close()

	// Drain the body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keep-alive returned status %d", resp.StatusCode)
	}

	return nil
}
