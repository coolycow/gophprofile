package model

import (
	"path"
	"regexp"
	"strconv"
)

// thumbnailSideFromKey регулярное выражение для извлечения размера миниатюры из ключа S3
var thumbnailSideFromKey = regexp.MustCompile(`_(\d+)\.[^/]+$`)

// ThumbnailSizeLabelFromS3Key по ключу вида .../uuid_100.jpg возвращает "100x100".
func ThumbnailSizeLabelFromS3Key(s3Key string) string {
	base := path.Base(s3Key)
	m := thumbnailSideFromKey.FindStringSubmatch(base)

	// Если не найдено совпадение, то возвращаем пустую строку
	if len(m) < 2 {
		return ""
	}

	n := m[1]

	// Если размер не является числом, то возвращаем пустую строку
	if _, err := strconv.Atoi(n); err != nil {
		return ""
	}

	return n + "x" + n
}
