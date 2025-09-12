package library

import (
	"seanime/internal/library/anime"
)

// LocalFileProcessor defines the interface for processing LocalFile changes
type LocalFileProcessor interface {
	ProcessNewLocalFile(lf *anime.LocalFile) error
	ProcessModifiedLocalFile(lf *anime.LocalFile) error
	ProcessRemovedLocalFile(filePath string, mediaID int) error
}