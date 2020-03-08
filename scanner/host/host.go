package host

import (
	"crypto/md5"
	"fmt"
	"io"
)

// Host struct host information about discovered network client
type Host struct {
	id  string
	IP  string
	MAC string
}

// ID will generate unique MD5 hash of host by his properties
// and cache generated hash for future usage
func (h *Host) ID() string {
	if h.id == "" {
		hash := md5.New()
		_, _ = io.WriteString(hash, h.IP+h.MAC)
		h.id = fmt.Sprintf("%x", hash.Sum(nil))
	}

	return h.id
}
