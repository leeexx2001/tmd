package downloader

import (
	pathpkg "path"
	"strings"
)

var validImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

func ExtractImageExtFromURL(url string) string {
	ext := pathpkg.Ext(url)
	ext = strings.ToLower(ext)

	if validImageExts[ext] {
		return ext
	}
	return ".jpg"
}
