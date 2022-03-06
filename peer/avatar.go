package peer

import (
	"fmt"
	"io"
	"strings"
)

type Avatar interface {
	UploadAvatar(data io.Reader) (string, error)

	DownloadAvatar(userAddress string) ([]byte, error)

	GetLocalhost() string
}

func AvatarName(address string) string {
	return fmt.Sprintf("AvatarFile-%v", address)
}

func IsAvatarName(filename string) bool {
	return strings.HasPrefix(filename, "AvatarFile-")
}
