package domain

type FAQ struct {
	ID         int64
	Pertanyaan string
	KataKunci  []string
	Jawaban    string
	Kategori   string
	Aktif      bool
}
