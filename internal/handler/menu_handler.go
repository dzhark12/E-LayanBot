package handler

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"elayan.local/desa/internal/gemini"
)

const menuText = "Selamat datang di layanan *E-LAYAN Desa Sabah Balau* 👋\n\n" +
	"Yang bisa kami bantu:\n" +
	"1. Ajukan surat → ketik *SURAT*\n" +
	"2. Cek status → kirim kode permohonan (contoh: *DM01245*)\n" +
	"3. Tanya informasi desa → tulis pertanyaan Anda (jam layanan, syarat, biaya, kontak)\n\n" +
	"Ketik *BATAL* kapan saja untuk membatalkan pengajuan.\n" +
	"Demi keamanan, jangan kirim NIK atau data pribadi lewat chat biasa; data hanya diminta lewat alur *SURAT*."

const statusNotFoundText = "Kode tidak ditemukan. Pastikan kode ditulis lengkap (contoh: DM01245) " +
	"dan dikirim dari nomor WhatsApp yang dipakai saat mengajukan. Ketik MENU untuk bantuan."

// identityText menjelaskan siapa bot ini tanpa membuka detail internal
// (model, instruksi sistem, infrastruktur) maupun data warga.
const identityText = "Saya *asisten otomatis E-LAYAN Desa Sabah Balau* 🤖, bukan petugas manusia.\n\n" +
	"Saya membalas secara otomatis untuk membantu pengajuan surat, cek status, dan informasi layanan desa " +
	"berdasarkan data yang disiapkan perangkat desa. Keputusan atas surat tetap dibuat oleh petugas desa.\n\n" +
	"Pertanyaan informasi dapat diproses dengan bantuan layanan AI, jadi jangan kirim NIK, kata sandi, kode OTP, " +
	"atau data pribadi lewat chat biasa. Saya tidak membagikan data pribadi warga.\n\n" +
	"Ketik *MENU* untuk melihat layanan."

const thanksText = "Sama-sama 🙏 Jika ada yang ingin ditanyakan lagi, ketik *MENU*."

// identityFact ikut dikirim ke asisten AI agar variasi pertanyaan tentang
// bot ("kamu bisa ngobrol?", "yang balas siapa?") tetap dijawab konsisten.
// Isinya publik dan tidak memuat detail internal.
var identityFact = gemini.Fact{
	Question: "Siapa kamu? Apakah ini bot atau manusia? Kenapa bisa membalas sendiri?",
	Answer: "Saya asisten otomatis E-LAYAN Desa Sabah Balau, bukan petugas manusia. Saya membantu pengajuan surat, " +
		"cek status, dan informasi layanan desa berdasarkan data dari perangkat desa; keputusan surat tetap oleh petugas. " +
		"Jangan kirim NIK, kata sandi, atau data pribadi lewat chat biasa. Ketik MENU untuk melihat layanan.",
}

// Batas ketat agar pencocokan murah dan tidak bisa disalahgunakan dengan
// pesan panjang; pesan yang lebih panjang diproses seperti biasa.
const (
	maxShortRunes = 80
	maxShortWords = 10
)

// greetings: sapaan pembuka yang menampilkan menu. Satu atau dua kata.
var greetings = map[string]bool{
	"menu": true, "halo": true, "hallo": true, "helo": true, "hello": true,
	"hai": true, "hi": true, "hei": true, "hay": true, "p": true, "permisi": true,
	"punten": true, "salam": true, "assalamualaikum": true, "asalamualaikum": true,
	"assalamu alaikum": true, "assalamu'alaikum": true, "pagi": true, "siang": true,
	"sore": true, "malam": true, "selamat pagi": true, "selamat siang": true,
	"selamat sore": true, "selamat malam": true,
}

// politeWords: kata sopan/pengisi yang boleh menyertai sapaan, ucapan
// terima kasih, atau pertanyaan identitas tanpa mengubah maknanya.
var politeWords = map[string]bool{
	"min": true, "admin": true, "kak": true, "ka": true, "pak": true, "bu": true,
	"bapak": true, "ibu": true, "mas": true, "mbak": true, "bang": true, "dek": true,
	"semua": true, "desa": true, "wr": true, "wb": true, "warahmatullahi": true,
	"wabarakatuh": true, "warahmatullah": true,
	"ya": true, "yah": true, "sih": true, "nih": true, "ni": true, "dong": true,
	"deh": true, "kah": true, "tuh": true,
}

// identityPhrases: pertanyaan tentang bot itu sendiri, dicocokkan utuh
// setelah kata sopan/pengisi dibuang.
var identityPhrases = map[string]bool{
	"kamu siapa": true, "siapa kamu": true, "anda siapa": true, "siapa anda": true,
	"ini siapa": true, "siapa ini": true, "ini apa": true, "apa ini": true,
	"dengan siapa": true, "sama siapa": true, "ini dengan siapa": true, "ini sama siapa": true,
	"bot": true, "robot": true, "chatbot": true, "ini bot": true, "kamu bot": true,
	"anda bot": true, "ini robot": true, "kamu robot": true, "ini chatbot": true,
	"kamu chatbot": true, "ini ai": true, "kamu ai": true, "ini manusia": true,
	"kamu manusia": true, "manusia atau bot": true, "bot atau manusia": true,
	"ini manusia atau bot": true, "ini bot atau manusia": true, "kamu manusia atau bot": true,
	"kamu bot atau manusia": true, "dengan siapa ini": true, "ini dengan siapa ya": true,
	"ini orang": true, "ini admin": true, "ini beneran admin": true, "ini otomatis": true,
	"kamu otomatis": true, "balas otomatis": true, "bales otomatis": true,
	"ini nomor apa": true, "nomor apa ini": true, "ini layanan apa": true,
	"layanan apa ini": true, "kok bisa bales sendiri": true, "kok bisa balas sendiri": true,
	"kok bales sendiri": true, "kok balas sendiri": true, "kok bisa bales otomatis": true,
	"kok bisa balas otomatis": true,
}

// thanksWords: pesan yang SELURUHNYA terdiri dari kata-kata ini dianggap
// ucapan terima kasih / penutup.
var thanksWords = map[string]bool{
	"terima": true, "kasih": true, "terimakasih": true, "makasih": true, "makasi": true,
	"thanks": true, "thank": true, "you": true, "thx": true, "tq": true, "ok": true,
	"oke": true, "okay": true, "okey": true, "sip": true, "siap": true, "baik": true,
	"mantap": true, "noted": true, "banyak": true,
}

// shortWords memecah pesan pendek menjadi kata huruf kecil. Pesan kosong,
// panjang, atau berangka (kode, NIK, nomor) ditolak agar ditangani jalur
// normal, termasuk penolakan NIK oleh asisten.
func shortWords(text string) []string {
	if text == "" || utf8.RuneCountInString(text) > maxShortRunes {
		return nil
	}
	if strings.ContainsFunc(text, unicode.IsDigit) {
		return nil
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	})
	if len(words) > maxShortWords {
		return nil
	}
	return words
}

// greetingPrefix mengembalikan jumlah kata sapaan di awal (0, 1, atau 2).
func greetingPrefix(words []string) int {
	switch {
	case len(words) >= 2 && greetings[words[0]+" "+words[1]]:
		return 2
	case len(words) >= 1 && greetings[words[0]]:
		return 1
	}
	return 0
}

func withoutPolite(words []string) []string {
	out := make([]string, 0, len(words))
	for _, word := range words {
		if !politeWords[word] {
			out = append(out, word)
		}
	}
	return out
}

// isGreeting hanya bernilai true bila SELURUH pesan berupa sapaan
// (opsional diikuti kata sopan). Kalimat yang berisi maksud lain, misalnya
// "selamat pagi, saya mau tanya syarat domisili", tidak dianggap sapaan
// sehingga tetap diteruskan ke FAQ/AI.
func isGreeting(text string) bool {
	words := shortWords(text)
	next := greetingPrefix(words)
	if next == 0 {
		return false
	}
	remaining := withoutPolite(words[next:])
	// Sapaan singkat dengan panggilan/nama seperti "hai Jidan" tetap
	// ditangani sebagai sapaan, bukan dikirim sebagai pertanyaan FAQ.
	return len(remaining) == 0 || len(remaining) == 1
}

// isIdentityQuestion bernilai true bila pesan (boleh diawali sapaan)
// seluruhnya berupa pertanyaan tentang bot, misalnya "halo, ini bot ya kak?".
func isIdentityQuestion(text string) bool {
	words := shortWords(text)
	if len(words) == 0 {
		return false
	}
	if next := greetingPrefix(words); next > 0 && next < len(words) {
		words = words[next:]
	}
	rest := withoutPolite(words)
	return len(rest) > 0 && identityPhrases[strings.Join(rest, " ")]
}

// isThanks bernilai true bila pesan seluruhnya ucapan terima kasih/penutup.
func isThanks(text string) bool {
	words := shortWords(text)
	rest := withoutPolite(words)
	if len(rest) == 0 {
		return false
	}
	for _, word := range rest {
		if !thanksWords[word] {
			return false
		}
	}
	return true
}
