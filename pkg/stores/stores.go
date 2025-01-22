package stores

import (
	"net/http"

	"github.com/KyleBrandon/scriptoria/pkg/stores/drive"
)

type Store interface {
	Initialize(mux *http.ServeMux) error
}

func BuildStore(storeName string) (Store, error) {
	switch storeName {
	case "Google Drive":
		return &drive.GoogleDrive{}, nil
		// case "Local":
		// 	return LocalStore{}
	}

	return nil, nil
}
