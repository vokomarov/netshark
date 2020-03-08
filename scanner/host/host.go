package host

import (
	"crypto/md5"
	"fmt"
	"io"
)

type Host struct {
	id  string
	IP  string
	MAC string
}

func (h *Host) ID() string {
	if h.id == "" {
		hash := md5.New()
		_, _ = io.WriteString(hash, h.IP+h.MAC)
		h.id = fmt.Sprintf("%x", hash.Sum(nil))
	}

	return h.id
}
