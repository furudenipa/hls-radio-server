package hls

import (
	"path/filepath"
	"strings"
)

// /srv/radio/contents/<category>/<id>/<file>
// -> /contents/<category>/<id>/<file>
func localToURL(localPath string) string {
	// パスを標準化
	cleanPath := filepath.ToSlash(filepath.Clean(localPath))
	const prefix = "/srv/radio"

	// プレフィックスで始まる場合は削除
	if strings.HasPrefix(cleanPath, prefix) {
		return strings.TrimPrefix(cleanPath, prefix)
	}

	// プレフィックスがない場合は、パスを維持
	return cleanPath
}
