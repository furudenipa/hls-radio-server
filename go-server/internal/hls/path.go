package hls

import (
	"strings"
)

// /srv/radio/contents/<category>/<id>/<file>
// -> /contents/<category>/<id>/<file>
func localToURL(localPath string) string {
	const prefix = "/srv/radio"
	if strings.HasPrefix(localPath, prefix) {
		return strings.TrimPrefix(localPath, prefix)
	}
	return localPath
}
