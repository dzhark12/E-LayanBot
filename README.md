# E-LAYAN Desa Sabah Balau

MVP pengajuan surat warga lewat WhatsApp. Warga memilih jenis surat, menjelaskan keperluan, menyetujui pemrosesan data, lalu mengirim foto persyaratan. Perangkat desa bekerja dari dashboard web: melihat foto privat, mengisi identitas secara manual, dan memperbarui status.

## Menjalankan pilot lokal

1. Salin `.env.example` menjadi `.env`.
2. Ganti password database, `BOT_API_TOKEN`, `WA_HASH_KEY`, dan password admin dengan secret yang kuat. Jangan ubah `WA_HASH_KEY` setelah data tersimpan.
3. Minta Sekdes menyetujui `ATTACHMENT_RETENTION_DAYS`, lalu isi nilainya. Aplikasi menolak berjalan selama keputusan retensi belum diisi.
4. Jalankan `docker compose up --build` dari terminal. Pada pemakaian pertama, pindai QR yang ditampilkan service `bot` dari WhatsApp → Perangkat tertaut. QR dapat dilihat lagi dengan `docker compose logs -f bot`.
5. Buka `http://localhost:8080` dan masuk dengan `ADMIN_USERNAME` serta `ADMIN_PASSWORD` dari `.env`.

Compose hanya membuka dashboard ke komputer lokal dan menyimpan PostgreSQL serta lampiran pada volume Docker privat. Untuk server, pasang domain dan reverse proxy HTTPS, lalu tetapkan `SESSION_COOKIE_SECURE=true`. Jangan publikasikan QR pairing atau secret di log yang dapat diakses umum.

## Alur yang tersedia

- Bot: `MENU`, `SURAT`, pemilihan jenis, ringkasan keperluan, persetujuan, foto bertakarir `KTP`, `KK`, `RT`, atau `LAINNYA`, dan konfirmasi `SELESAI`. `BATAL` membatalkan sesi. Sesi dan foto sementara kedaluwarsa setelah 24 jam.
- Jenis awal: `DM`, `SU`, `TM`, dan `PK`. Syarat yang ditampilkan berasal dari data jenis surat yang dikelola di dashboard.
- Dashboard: daftar dan detail pengajuan, viewer lampiran, input identitas manual (nama, tempat/tanggal lahir, jenis kelamin, agama, NIK, pekerjaan, alamat), perubahan status dan riwayat, serta pengelolaan FAQ dan jenis surat pada `/settings`.
- Bot: cek kode hanya mengungkap status kepada nomor WhatsApp pemohon yang cocok. Saat status menjadi `Selesai`, service bot mengirim notifikasi otomatis.
- FAQ hanya menjawab dari data yang dimasukkan admin; LLM tidak digunakan.

## Penyimpanan dan API internal

Semua endpoint bot memakai `Authorization: Bearer $BOT_API_TOKEN`. `POST /api/v1/submissions` menerima `multipart/form-data` berisi `request_id` (UUID idempotensi), `letter_code`, `wa_jid`, `summary`, `consent_version=v1`, serta pasangan `attachments` dan `attachment_types`. Maksimal lima foto JPEG/PNG, 10 MB per foto. Foto dipindah ke storage privat dan hanya dapat dilihat setelah login dashboard.

API menyimpan JID untuk balasan/notifikasi dan HMAC nomor untuk cek status; pertanyaan status dibatasi sepuluh kali per jam per nomor. Pertahankan `WA_HASH_KEY` selama data tersebut masih dipakai. Data lampiran permohonan yang sudah selesai dihapus setelah masa retensi yang disetujui.

## Batas MVP dan keputusan yang diperlukan

- Template resmi dan syarat dari desa belum tersedia. Karena itu pembuatan/pratinjau PDF belum diaktifkan; perangkat desa masih perlu memindahkan data ke template resmi secara manual setelah melihat foto.
- Sebelum penggunaan warga, desa perlu menetapkan template dan syarat tiap surat, akun petugas, retensi lampiran, serta penanggung biaya domain/hosting. Data awal FAQ dan persyaratan juga perlu diisi.
- Pilot WhatsApp memakai `whatsmeow` dan memerlukan nomor khusus yang ditautkan sebagai perangkat. Ketersediaan serta stabilitas akun WhatsApp perlu dipantau selama pilot.
- Backup terjadwal dan statistik evaluasi SDGs belum dikonfigurasi. Jangan menganggap volume Docker sebagai backup.
