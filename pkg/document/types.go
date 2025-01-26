package document

type (
	Document struct {
		ID   string
		Name string
	}

	DocumentStorage interface {
		Initialize() error
	}
)
