package domain

import (
	"encoding/json"
	"time"
)

// StreamOracleResponse takes a response, marshals it, and streams it chunk-by-chunk to tokenChan.
// It uses its own independent background context so it is never cancelled by the HTTP request's
// context timeout (which fires via defer cancel() when queryOracleCmd returns).
func StreamOracleResponse(res *OracleResponse, tokenChan chan<- string) {
	data, _ := json.MarshalIndent(res, "", "  ")
	strData := string(data)

	go func() {
		defer close(tokenChan)
		runes := []rune(strData)
		chunkSize := 8
		for i := 0; i < len(runes); i += chunkSize {
			end := i + chunkSize
			if end > len(runes) {
				end = len(runes)
			}
			tokenChan <- string(runes[i:end])
			time.Sleep(2 * time.Millisecond)
		}
	}()
}
