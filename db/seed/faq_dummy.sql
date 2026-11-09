-- FAQ seed kurasi E-LAYAN Desa Sabah Balau.
-- 70 entri total: 60 jawaban aman berbasis fungsi bot dan 10 entri
-- berstatus PERLU KONFIRMASI DESA (nonaktif sampai datanya disahkan).
-- Tidak mengandung nama, nomor pribadi, jadwal, biaya, atau SLA rekaan.
--
-- Jalankan dari folder proyek dengan PowerShell:
--   Get-Content db/seed/faq_dummy.sql -Raw | docker compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
--
-- Skrip mengganti FAQ seed lama yang judulnya cocok persis dan aman
-- dijalankan ulang sebelum petugas mengedit seed ini melalui dashboard.
-- FAQ lain yang tidak cocok dengan judul seed tidak dihapus.
-- Entri konfirmasi desa sengaja nonaktif agar tidak dijawab bot/AI.
BEGIN;

CREATE TEMP TABLE faq_seed (
  n INTEGER PRIMARY KEY,
  pertanyaan TEXT NOT NULL,
  kata_kunci TEXT[] NOT NULL,
  jawaban TEXT NOT NULL,
  kategori TEXT NOT NULL,
  aktif BOOLEAN NOT NULL
) ON COMMIT DROP;

INSERT INTO faq_seed (n, pertanyaan, kata_kunci, jawaban, kategori, aktif) VALUES
(1, 'Bisakah saya meminta NIK, nomor KK, alamat, atau data warga lain?', ARRAY['nik warga','nomor kk warga','alamat warga','data pribadi warga','data penduduk','daftar penerima bansos'], 'Maaf, bot tidak memberikan data pribadi warga lain. Jangan meminta atau membagikan NIK, nomor KK, alamat lengkap, nomor telepon, maupun daftar penerima bantuan lewat chat.', 'Privasi & Keamanan', TRUE),
(2, 'Data pribadi apa yang tidak boleh saya kirim lewat chat biasa?', ARRAY['kirim data pribadi','kirim nik','kirim nomor kk','alamat lengkap','foto ktp chat','foto kk chat'], 'Jangan kirim NIK, nomor KK, alamat lengkap, kata sandi, OTP, atau data pribadi orang lain dalam pertanyaan biasa. Untuk foto persyaratan, mulai alur SURAT dan ikuti arahan bot.', 'Privasi & Keamanan', TRUE),
(3, 'Apakah petugas akan meminta kata sandi atau kode OTP?', ARRAY['minta password','minta kata sandi','minta otp','kode otp','pin atm'], 'Jangan berikan kata sandi, PIN, atau kode OTP melalui chat. Bot pengajuan surat tidak memerlukan data tersebut. Jika ada pihak yang memintanya atas nama desa, verifikasi melalui kanal resmi yang diumumkan pemerintah desa.', 'Privasi & Keamanan', TRUE),
(4, 'Bolehkah saya mengirim foto KTP atau KK sebelum mengetik SURAT?', ARRAY['foto ktp sebelum surat','kirim ktp di chat','kirim kk di chat','dokumen lewat chat biasa'], 'Jangan kirim foto dokumen dalam chat biasa. Ketik SURAT dan kirim foto hanya saat bot meminta lampiran di alur pengajuan.', 'Privasi & Keamanan', TRUE),
(5, 'Apakah saya berbicara dengan petugas manusia?', ARRAY['siapa kamu','ini bot','ini manusia','petugas manusia','asisten otomatis'], 'Saya asisten otomatis E-LAYAN. Saya membantu alur pengajuan, pengecekan status, dan informasi yang tersedia. Keputusan atas permohonan tetap dibuat petugas desa.', 'Tentang Bot', TRUE),
(6, 'Apakah bot memakai AI untuk menjawab pertanyaan?', ARRAY['pakai ai','pakai gemini','jawaban ai','asisten ai','pertanyaan diproses ai'], 'Pertanyaan yang tidak cocok dengan FAQ dapat diproses oleh asisten AI jika fitur itu aktif. Jawaban dibatasi pada FAQ yang tersedia. Jangan masukkan data pribadi dalam pertanyaan.', 'Tentang Bot', TRUE),
(7, 'Apa yang dapat saya lakukan lewat bot E-LAYAN?', ARRAY['layanan bot','bisa apa','fungsi bot','layanan elayan'], 'Bot dapat memandu pengajuan jenis surat yang tersedia di menu, memberi kode permohonan, membantu cek status, dan menjawab informasi layanan yang sudah dimasukkan ke FAQ.', 'Layanan Bot', TRUE),
(8, 'Surat apa saja yang bisa diajukan lewat bot?', ARRAY['jenis surat bot','surat lewat bot','daftar jenis surat','surat yang tersedia'], 'Ketik SURAT untuk melihat daftar jenis yang sedang aktif. Daftar dan persyaratan yang tampil di alur itu menjadi acuan pengajuan saat ini.', 'Layanan Surat', TRUE),
(9, 'Bagaimana cara memulai pengajuan surat?', ARRAY['mulai pengajuan','cara mengajukan surat','ajukan surat','buat surat lewat bot'], 'Ketik SURAT, pilih jenis surat dengan nomor atau kode, jelaskan keperluan, lalu ikuti instruksi bot sampai pengajuan dikonfirmasi.', 'Layanan Surat', TRUE),
(10, 'Bagaimana memilih jenis surat di menu bot?', ARRAY['pilih jenis surat','nomor jenis surat','kode surat'], 'Saat daftar surat tampil, balas dengan nomor atau kode jenis surat yang dipilih. Jika bingung, ketik MENU untuk melihat petunjuk awal.', 'Layanan Surat', TRUE),
(11, 'Di mana saya melihat persyaratan surat?', ARRAY['syarat jenis surat','persyaratan surat','dokumen yang diperlukan','syarat dokumen'], 'Ketik SURAT lalu pilih jenis surat. Bot akan menampilkan persyaratan yang sedang dikonfigurasi untuk jenis tersebut. Persyaratan akhir tetap mengikuti verifikasi petugas desa.', 'Layanan Surat', TRUE),
(12, 'Apa yang perlu ditulis pada bagian keperluan surat?', ARRAY['isi keperluan','ringkasan keperluan','menulis keperluan'], 'Tuliskan keperluan surat secara singkat dan jelas, tanpa menambahkan data sensitif yang tidak diminta. Bot menerima ringkasan 1 sampai 500 karakter.', 'Layanan Surat', TRUE),
(13, 'Mengapa bot meminta persetujuan sebelum menerima dokumen?', ARRAY['tujuan persetujuan','kenapa setuju','persetujuan data','setuju pemrosesan'], 'Persetujuan diminta sebelum alur menerima foto persyaratan untuk pengajuan. Baca instruksi pemrosesan data yang ditampilkan bot sebelum mengetik SETUJU.', 'Privasi & Keamanan', TRUE),
(14, 'Apa yang terjadi jika saya tidak menyetujui pemrosesan data?', ARRAY['tidak setuju','menolak persetujuan','tidak mau setuju'], 'Tanpa SETUJU, pengajuan tidak dapat dilanjutkan ke tahap pengiriman foto. Ketik BATAL untuk menghentikan sesi pengajuan.', 'Privasi & Keamanan', TRUE),
(15, 'Format foto apa yang diterima bot?', ARRAY['format foto','jpeg','jpg','png','foto tidak diterima'], 'Bot menerima foto JPEG atau PNG yang valid. Kirim sebagai foto dokumen sesuai instruksi pada alur pengajuan.', 'Layanan Surat', TRUE),
(16, 'Berapa ukuran maksimal satu foto persyaratan?', ARRAY['ukuran foto','maksimal 10 mb','ukuran lampiran'], 'Maksimal 10 MB untuk setiap foto. Gunakan JPEG atau PNG.', 'Layanan Surat', TRUE),
(17, 'Berapa banyak foto yang dapat dikirim untuk satu pengajuan?', ARRAY['jumlah foto','maksimal lima foto','batas lampiran'], 'Maksimal 5 foto untuk satu pengajuan. Bot meminta setidaknya 1 foto sebelum pengajuan dapat dikirim.', 'Layanan Surat', TRUE),
(18, 'Apa caption untuk foto persyaratan?', ARRAY['caption foto','label foto','jenis lampiran','caption ktp'], 'Tambahkan caption sesuai kategori pada setiap foto: KTP, KK, RT, atau LAINNYA.', 'Layanan Surat', TRUE),
(19, 'Mengapa foto persyaratan saya belum diterima?', ARRAY['foto ditolak bot','foto belum diterima','caption salah','foto jpeg png'], 'Periksa bahwa foto berformat JPEG atau PNG, ukurannya tidak lebih dari 10 MB, dan memiliki caption KTP, KK, RT, atau LAINNYA. Jika masih gagal, coba kirim ulang foto yang jelas dan valid.', 'Layanan Surat', TRUE),
(20, 'Apakah saya bisa mengirim SELESAI tanpa foto?', ARRAY['selesai tanpa foto','minimal foto','belum kirim lampiran'], 'Tidak. Bot meminta minimal satu foto persyaratan sebelum menerima perintah SELESAI.', 'Layanan Surat', TRUE),
(21, 'Kapan saya harus mengetik SELESAI?', ARRAY['kapan selesai','sudah kirim foto','kirim pengajuan'], 'Setelah semua foto persyaratan terkirim, ketik SELESAI untuk mengirim pengajuan.', 'Layanan Surat', TRUE),
(22, 'Apa bukti bahwa pengajuan saya berhasil dikirim?', ARRAY['bukti pengajuan','kode permohonan','pengajuan diterima'], 'Jika server mengonfirmasi pengajuan, bot membalas dengan kode permohonan dan status awal. Simpan kode tersebut untuk mengecek status.', 'Layanan Surat', TRUE),
(23, 'Bagaimana jika belum menerima kode permohonan?', ARRAY['belum dapat kode','kode belum diterima','tidak ada konfirmasi'], 'Periksa balasan bot. Jika bot menyatakan belum mendapat konfirmasi server, ketik SELESAI lagi untuk mencoba ulang. Sesi dan foto sementara berlaku 24 jam; jangan mengirim ulang dokumen di chat biasa.', 'Layanan Surat', TRUE),
(24, 'Bagaimana cara mengecek status pengajuan?', ARRAY['cek status surat','status permohonan','cek pengajuan','status surat saya'], 'Kirim kode permohonan lengkap ke bot dari nomor WhatsApp yang digunakan saat pengajuan.', 'Layanan Surat', TRUE),
(25, 'Mengapa status hanya bisa dicek dari nomor WhatsApp pemohon?', ARRAY['nomor yang sama','nomor whatsapp pemohon','cek dari nomor lain'], 'Pengecekan mencocokkan kode permohonan dengan nomor WhatsApp pengaju untuk melindungi informasi permohonan. Bot tidak menampilkan status jika nomor tidak cocok.', 'Privasi & Keamanan', TRUE),
(26, 'Status apa saja yang mungkin tampil?', ARRAY['arti status','status menunggu','status dikerjakan','perlu perbaikan','status ditolak','status selesai'], 'Status yang digunakan sistem: Menunggu, Dikerjakan, Perlu Perbaikan, Selesai, dan Ditolak. Jika status menunjukkan Perlu Perbaikan atau Ditolak, ikuti arahan resmi petugas desa; bot belum menampilkan rincian tindakan koreksi.', 'Layanan Surat', TRUE),
(27, 'Apa arti kode permohonan seperti DM01245?', ARRAY['format kode','kode dm','kode permohonan artinya'], 'Kode permohonan adalah nomor rujukan untuk mengecek status. Ketik persis seperti yang dikirim bot dan simpan secara pribadi.', 'Layanan Surat', TRUE),
(28, 'Apa yang harus dilakukan jika kode tidak ditemukan?', ARRAY['kode tidak ditemukan','kode salah','status tidak ditemukan'], 'Periksa kembali kode dan kirim dari nomor WhatsApp yang dipakai saat mengajukan. Bot memakai balasan yang sama jika kode tidak cocok atau nomor berbeda demi menjaga privasi.', 'Privasi & Keamanan', TRUE),
(29, 'Bagaimana jika saya lupa atau kehilangan kode permohonan?', ARRAY['lupa kode','kehilangan kode','kode hilang'], 'Periksa kembali pesan konfirmasi pengajuan. Jangan kirim kode atau data identitas ke orang yang tidak dikenal. Jika tetap tidak menemukannya, gunakan kanal resmi desa setelah informasinya tersedia.', 'Layanan Surat', TRUE),
(30, 'Apa fungsi BATAL saat saya sedang mengajukan surat?', ARRAY['batal pengajuan','hentikan sesi','batalkan sesi'], 'Ketik BATAL untuk menghentikan sesi pengajuan yang masih berlangsung dan menghapus foto sementara yang belum dikirim sebagai permohonan.', 'Layanan Surat', TRUE),
(31, 'Apakah BATAL membatalkan permohonan yang sudah mendapat kode?', ARRAY['batal setelah kirim','batalkan kode','batalkan permohonan terkirim'], 'Tidak. BATAL menghentikan sesi pengajuan yang belum dikirim. Bot belum menyediakan pembatalan permohonan yang sudah mendapat kode; prosedur untuk itu perlu dikonfirmasi ke desa.', 'Layanan Surat', TRUE),
(32, 'Berapa lama sesi pengajuan yang belum selesai tersimpan?', ARRAY['sesi 24 jam','masa sesi','foto sementara'], 'Sesi percakapan dan foto sementara kedaluwarsa setelah 24 jam. Selesaikan pengajuan sebelum masa tersebut berakhir.', 'Privasi & Keamanan', TRUE),
(33, 'Apakah petugas dapat melihat foto persyaratan?', ARRAY['siapa melihat foto','petugas melihat dokumen','akses lampiran'], 'Foto persyaratan pengajuan ditampilkan kepada petugas yang masuk ke dashboard untuk memeriksa permohonan. Jangan mengirim dokumen lewat chat biasa.', 'Privasi & Keamanan', TRUE),
(34, 'Apakah bot dapat memastikan surat saya akan diterbitkan?', ARRAY['jaminan surat','pasti diterima','surat pasti jadi'], 'Tidak. Bot membantu mengirim permohonan dan mengecek status. Pemeriksaan dan keputusan tetap dilakukan petugas desa.', 'Layanan Surat', TRUE),
(35, 'Bisakah bot mengurus surat yang tidak muncul di menu SURAT?', ARRAY['surat tidak ada di menu','surat lain','layanan tidak tersedia'], 'Bot hanya menerima jenis surat yang tampil di menu saat pengajuan. Untuk layanan lain, bot belum dapat menjanjikan prosesnya; gunakan kanal resmi desa setelah tersedia.', 'Layanan Surat', TRUE),
(36, 'Apakah bot memberikan daftar penerima bantuan atau data warga?', ARRAY['daftar penerima bantuan','data penerima bansos','data warga lain'], 'Tidak. Bot tidak memberikan daftar penerima bantuan, NIK, alamat, nomor telepon, atau data pribadi warga lain.', 'Privasi & Keamanan', TRUE),
(37, 'Bagaimana mengenali pesan atau nomor palsu yang mengatasnamakan desa?', ARRAY['penipuan mengatasnamakan desa','nomor palsu','minta transfer','tautan mencurigakan'], 'Jangan berikan OTP, kata sandi, atau data pribadi dan jangan transfer ke rekening pribadi tanpa verifikasi. Cocokkan informasi hanya dengan kanal resmi desa yang diumumkan pemerintah desa.', 'Privasi & Keamanan', TRUE),
(38, 'Bisakah saya mengirim pengaduan jalan rusak lewat bot?', ARRAY['lapor jalan rusak','pengaduan lewat bot','keluhan fasilitas'], 'Bot ini belum memiliki alur untuk menerima dan melacak pengaduan fasilitas umum. Jangan kirim data pribadi orang lain; kanal pengaduan resmi perlu dikonfirmasi ke desa.', 'Layanan Masyarakat', TRUE),
(39, 'Apakah bot bisa menentukan kelayakan bantuan sosial?', ARRAY['layak bansos','penerima pkh','dapat blt','bantuan sosial saya'], 'Tidak. Bot tidak menilai kelayakan atau menjamin penerimaan bantuan sosial, dan tidak membagikan data penerima. Periksa informasi melalui kanal resmi program atau pemerintah desa.', 'Layanan Masyarakat', TRUE),
(40, 'Bisakah saya meminta bot memberi nasihat kesehatan pribadi?', ARRAY['diagnosis kesehatan','keluhan kesehatan pribadi','obat apa'], 'Bot E-LAYAN bukan layanan tenaga kesehatan dan tidak dapat menilai kondisi pribadi. Jangan kirim data kesehatan pribadi ke chat ini; hubungi tenaga kesehatan atau fasilitas kesehatan.', 'Layanan Masyarakat', TRUE),
(41, 'Apa yang harus dilakukan jika bot tidak memahami pertanyaan?', ARRAY['bot tidak paham','jawaban tidak sesuai','informasi belum tersedia'], 'Coba tulis pertanyaan singkat tentang layanan desa atau ketik MENU. Bot hanya dapat menjawab informasi yang tersedia dan tidak boleh menebak biaya, syarat, jadwal, atau keputusan.', 'Layanan Bot', TRUE),
(42, 'Bagaimana jika saya ingin bicara dengan petugas?', ARRAY['bicara dengan petugas','hubungi petugas','kontak petugas desa'], 'Bot adalah asisten otomatis. Untuk berbicara dengan petugas, gunakan kanal resmi desa yang telah diumumkan. Jangan mengirim data pribadi sebelum yakin kanal tersebut resmi.', 'Layanan Masyarakat', TRUE),
(43, '[KONFIRMASI DESA] Kapan jam pelayanan kantor desa?', ARRAY['jam pelayanan kantor','jam buka desa','jam kerja desa'], 'PERLU KONFIRMASI DESA: isi hari, jam, waktu istirahat, dan pengecualian hari libur sebelum FAQ diaktifkan.', 'PERLU KONFIRMASI DESA', FALSE),
(44, '[KONFIRMASI DESA] Di mana alamat dan lokasi resmi kantor desa?', ARRAY['alamat kantor desa','lokasi kantor desa','peta kantor desa'], 'PERLU KONFIRMASI DESA: isi alamat lengkap dan tautan/pin peta resmi yang boleh dipublikasikan.', 'PERLU KONFIRMASI DESA', FALSE),
(45, '[KONFIRMASI DESA] Apa kanal kontak resmi untuk warga?', ARRAY['nomor resmi desa','telepon kantor desa','kontak resmi desa','email resmi desa'], 'PERLU KONFIRMASI DESA: cantumkan hanya nomor atau kanal layanan resmi yang disetujui untuk publikasi; jangan gunakan nomor pribadi tanpa izin.', 'PERLU KONFIRMASI DESA', FALSE),
(46, '[KONFIRMASI DESA] Apakah setiap jenis surat memiliki biaya?', ARRAY['biaya surat desa','tarif surat','biaya layanan'], 'PERLU KONFIRMASI DESA: tetapkan informasi biaya per jenis surat dan sumber kebijakan resminya. Jangan aktifkan jawaban contoh atau perkiraan.', 'PERLU KONFIRMASI DESA', FALSE),
(47, '[KONFIRMASI DESA] Berapa lama proses tiap jenis surat?', ARRAY['estimasi surat','lama proses surat','target layanan'], 'PERLU KONFIRMASI DESA: tetapkan target waktu per jenis surat, kapan penghitungan dimulai, dan kondisi yang dapat memperpanjang proses.', 'PERLU KONFIRMASI DESA', FALSE),
(48, '[KONFIRMASI DESA] Bagaimana warga menerima surat setelah status Selesai?', ARRAY['ambil surat selesai','surat dikirim','pengambilan surat'], 'PERLU KONFIRMASI DESA: jelaskan apakah surat diambil, dikirim, atau diserahkan dengan cara lain, termasuk dokumen verifikasi yang diperlukan.', 'PERLU KONFIRMASI DESA', FALSE),
(49, '[KONFIRMASI DESA] Apa syarat final dan siapa yang boleh mengajukan tiap jenis surat?', ARRAY['syarat resmi surat','kelayakan domisili','surat diwakilkan'], 'PERLU KONFIRMASI DESA: validasi syarat, wilayah/kelayakan pemohon, dan aturan pengajuan oleh wakil untuk setiap jenis surat.', 'PERLU KONFIRMASI DESA', FALSE),
(50, '[KONFIRMASI DESA] Ke mana warga mengurus koreksi, penolakan, pembatalan, atau permintaan data?', ARRAY['status perlu perbaikan','status ditolak','hapus data permohonan','batalkan permohonan terkirim'], 'PERLU KONFIRMASI DESA: tetapkan kanal dan prosedur untuk status Perlu Perbaikan/Ditolak, pembatalan setelah pengajuan terkirim, serta permintaan akses atau penghapusan data.', 'PERLU KONFIRMASI DESA', FALSE),
(51, 'Kalau status surat sudah Selesai, bagaimana cara menerima atau mengambilnya?', ARRAY['surat selesai ambil dimana','cara menerima surat selesai','pengambilan surat selesai','surat jadi diambil','ambil lewat desa','harus ambil di desa','cara ambil surat'], 'Cek status dengan mengirim kode permohonan dari nomor WhatsApp yang dipakai saat mengajukan. Status Selesai berarti status permohonan telah diperbarui di sistem, tetapi FAQ ini belum memuat apakah surat diambil, dikirim, atau diserahkan dengan cara lain. Untuk memastikan cara dan waktunya, tanyakan melalui kanal resmi pemerintah desa; jangan kirim kode atau dokumen ke pihak yang tidak dikenal.', 'Layanan Surat', TRUE),
(52, 'Apakah surat yang diajukan lewat bot dikirim digital atau harus diambil?', ARRAY['surat digital','surat dikirim lewat wa','surat fisik','surat harus diambil'], 'Cara penyerahan surat belum ditetapkan dalam informasi yang tersedia. Setelah status berubah menjadi Selesai, konfirmasikan cara penerimaan melalui kanal resmi pemerintah desa.', 'Layanan Surat', TRUE),
(53, 'Apa arti status Selesai pada permohonan surat?', ARRAY['arti status selesai','status selesai surat','surat selesai berarti apa'], 'Selesai adalah salah satu status yang dapat tampil untuk permohonan. Bot tidak dapat menyimpulkan dari status itu saja apakah surat perlu diambil atau akan dikirim; tanyakan cara penyerahannya melalui kanal resmi pemerintah desa.', 'Layanan Surat', TRUE),
(54, 'Berapa lama sampai surat selesai diproses?', ARRAY['berapa lama surat jadi','estimasi surat saya','kapan surat selesai','lama proses permohonan'], 'Bot belum memiliki estimasi waktu proses yang disahkan untuk tiap jenis surat. Anda dapat mengecek status dengan mengirim kode permohonan dari nomor WhatsApp yang dipakai saat mengajukan; untuk perkiraan waktu, tanyakan melalui kanal resmi pemerintah desa.', 'Layanan Surat', TRUE),
(55, 'Bisakah pengajuan surat dipercepat atau diprioritaskan?', ARRAY['surat dipercepat','minta prioritas surat','proses surat urgent','surat mendesak'], 'Bot tidak dapat mengubah urutan atau menjanjikan percepatan proses. Sampaikan kebutuhan Anda melalui kanal resmi pemerintah desa tanpa mengirim data sensitif di chat biasa.', 'Layanan Surat', TRUE),
(56, 'Apakah pengajuan surat lewat bot berbayar?', ARRAY['biaya pengajuan surat','tarif bikin surat','bayar surat online','surat gratis'], 'Informasi biaya yang berlaku belum tersedia di FAQ ini. Jangan mengirim uang ke rekening pribadi berdasarkan pesan yang belum terverifikasi; konfirmasikan biaya melalui kanal resmi pemerintah desa.', 'Layanan Surat', TRUE),
(57, 'Bisakah saya mengubah data atau lampiran setelah permohonan dikirim?', ARRAY['ubah pengajuan terkirim','koreksi data surat','ganti lampiran surat','edit permohonan'], 'Bot belum menyediakan fitur untuk mengubah permohonan yang sudah mendapat kode. Minta arahan melalui kanal resmi pemerintah desa dan jangan kirim ulang dokumen pribadi sebelum diminta lewat alur resmi.', 'Layanan Surat', TRUE),
(58, 'Bagaimana membatalkan pengajuan yang sudah mendapat kode?', ARRAY['batalkan surat terkirim','batal permohonan berkode','cancel pengajuan terkirim'], 'Perintah BATAL hanya menghentikan sesi yang belum dikirim. Pembatalan permohonan yang sudah mendapat kode belum tersedia melalui bot; tanyakan prosedurnya melalui kanal resmi pemerintah desa.', 'Layanan Surat', TRUE),
(59, 'Apa yang harus saya lakukan jika statusnya Perlu Perbaikan?', ARRAY['status perlu perbaikan harus apa','perbaiki surat','dokumen perlu diperbaiki'], 'Ikuti arahan yang disampaikan petugas melalui kanal resmi. Bot belum memuat rincian dokumen koreksi untuk setiap permohonan; jangan mengirim dokumen atau data pribadi ke chat biasa.', 'Layanan Surat', TRUE),
(60, 'Apa yang harus dilakukan jika permohonan surat ditolak?', ARRAY['surat ditolak harus bagaimana','permohonan ditolak','mengajukan ulang surat ditolak'], 'Bot tidak dapat menetapkan alasan atau langkah banding dari status saja. Tanyakan alasan dan langkah berikutnya melalui kanal resmi pemerintah desa; keputusan permohonan dibuat petugas.', 'Layanan Surat', TRUE),
(61, 'Bolehkah keluarga mengambil surat atau mengurus permohonan sebagai wakil?', ARRAY['surat diambil keluarga','pengajuan diwakilkan','ambil surat diwakilkan','wakil pemohon'], 'Aturan perwakilan dan pengambilan oleh orang lain belum tersedia di FAQ ini. Konfirmasikan syaratnya melalui kanal resmi pemerintah desa sebelum mengirim dokumen atau meminta orang lain mengambil surat.', 'Layanan Surat', TRUE),
(62, 'Apakah saya bisa mengecek status permohonan orang lain?', ARRAY['cek status orang lain','status surat keluarga','permohonan warga lain'], 'Bot hanya mencocokkan kode dengan nomor WhatsApp yang digunakan saat pengajuan. Bot tidak memberikan status permohonan orang lain; pemohon sebaiknya mengecek sendiri dari nomor yang terdaftar.', 'Privasi & Keamanan', TRUE),
(63, 'Bolehkah membagikan kode permohonan di grup atau ke orang lain?', ARRAY['bagikan kode permohonan','kode surat di grup','orang lain tahu kode pengajuan'], 'Simpan kode permohonan secara pribadi dan bagikan hanya bila diperlukan kepada pihak resmi yang sudah Anda verifikasi. Jangan kirim kode bersama NIK, nomor KK, atau dokumen ke grup publik.', 'Privasi & Keamanan', TRUE),
(64, 'Apakah aman mengirim foto persyaratan di chat ini?', ARRAY['aman kirim foto persyaratan','kirim dokumen ke bot','unggah ktp ke bot'], 'Kirim foto hanya setelah memulai alur SURAT dan bot meminta lampiran, setelah membaca persetujuan pemrosesan data. Jangan mengirim dokumen dalam chat informasi biasa atau kepada akun yang belum diverifikasi.', 'Privasi & Keamanan', TRUE),
(65, 'Apakah bot bisa memberi tahu status atau keputusan surat dari nama saya?', ARRAY['cek surat pakai nama','status berdasarkan nama','surat saya disetujui belum'], 'Tidak. Bot tidak mencari permohonan berdasarkan nama dan tidak dapat menjanjikan keputusan. Untuk cek status, kirim kode permohonan dari nomor WhatsApp yang dipakai saat mengajukan.', 'Privasi & Keamanan', TRUE),
(66, 'Saya butuh layanan desa selain jenis surat di menu, bagaimana?', ARRAY['layanan desa lain','urus layanan selain surat','layanan tidak ada di menu'], 'Bot hanya memproses jenis surat yang tampil saat Anda mengetik SURAT. Untuk layanan lain, tanyakan apakah tersedia melalui kanal resmi pemerintah desa; bot tidak dapat mengirim pengaduan atau mengurus layanan di luar menu.', 'Layanan Masyarakat', TRUE),
(67, 'Bagaimana mengetahui apakah alamat saya termasuk wilayah desa ini?', ARRAY['wilayah desa saya','alamat masuk desa','batas wilayah desa','domisili wilayah desa'], 'Bot belum memiliki data untuk memastikan cakupan wilayah berdasarkan alamat. Jangan kirim alamat lengkap ke chat; tanyakan cakupan wilayah melalui kanal resmi pemerintah desa.', 'Informasi Desa', TRUE),
(68, 'Di mana saya melihat informasi resmi tentang program atau kegiatan desa?', ARRAY['program desa','kegiatan desa','informasi kegiatan warga','agenda desa'], 'Jadwal dan rincian program atau kegiatan belum tersedia di FAQ ini. Gunakan pengumuman atau kanal resmi yang diterbitkan pemerintah desa; bot tidak akan menebak jadwal maupun kelayakan peserta.', 'Informasi Desa', TRUE),
(69, '[KONFIRMASI DESA] Siapa perangkat desa dan apa tugas masing-masing?', ARRAY['nama perangkat desa','struktur pemerintahan desa','kepala desa siapa','perangkat desa'], 'PERLU KONFIRMASI DESA: masukkan nama/jabatan yang boleh dipublikasikan, uraian tugas, dan tanggal pembaruan sumber resmi sebelum FAQ diaktifkan.', 'PERLU KONFIRMASI DESA', FALSE),
(70, '[KONFIRMASI DESA] Apa fasilitas umum, program, dan kegiatan warga yang tersedia?', ARRAY['fasilitas umum desa','program desa resmi','kegiatan desa kapan','layanan posyandu'], 'PERLU KONFIRMASI DESA: masukkan hanya daftar fasilitas/program yang aktif, jadwal dan persyaratan yang sudah diverifikasi, serta sumber dan tanggal pembaruannya.', 'PERLU KONFIRMASI DESA', FALSE);

-- Hapus hanya seed lama yang dikenal dan baris seed ini saat skrip diulang.
DELETE FROM faq WHERE pertanyaan IN (
  'Bisakah saya meminta NIK, nomor KK, atau alamat warga lain?',
  'Bolehkah saya mengirim NIK, foto KTP/KK, kata sandi, atau kode OTP lewat chat ini?',
  'Bagaimana struktur pemerintahan Desa Sabah Balau?',
  'Siapa kepala desa Sabah Balau?',
  'Siapa sekretaris desa?',
  'Siapa yang mengurus pelayanan surat dan apa tugas Kaur serta Kasi?',
  'Siapa kepala dusun dan ketua RT saya?',
  'Apa itu BPD dan siapa ketuanya?',
  'Lembaga apa saja yang ada di desa?',
  'Saya mahasiswa atau pendatang yang tinggal di kontrakan. Bisakah membuat surat domisili?',
  'Bagaimana mengurus surat pindah masuk ke Desa Sabah Balau?',
  'Bagaimana prosedur surat pindah keluar dari desa?',
  'Bagaimana membuat surat keterangan belum menikah?',
  'Bagaimana cara mengurus surat pengantar nikah?',
  'Apakah desa bisa membuat surat kehilangan?',
  'Bagaimana cara mengurus surat keterangan kematian?',
  'Bagaimana cara membuat surat keterangan domisili?',
  'Bagaimana cara membuat Surat Keterangan Usaha (SKU)?',
  'Bagaimana cara mendapatkan Surat Keterangan Tidak Mampu (SKTM)?',
  'Bagaimana cara mengurus akta kelahiran anak?',
  'Bagaimana cara mengurus perubahan KTP atau Kartu Keluarga?',
  'Apakah untuk SKCK perlu surat pengantar dari desa?',
  'Bagaimana izin atau pengantar untuk mengadakan acara?',
  'Apakah kantor desa melayani legalisasi dokumen?',
  'Apakah pengurusan surat bisa diwakilkan anggota keluarga?',
  'Surat apa saja yang bisa dibuat di Desa Sabah Balau?',
  'Bagaimana cara mengajukan surat?',
  'Dokumen apa yang biasanya diperlukan untuk membuat surat?',
  'Berapa lama surat diproses?',
  'Apakah pengurusan surat dikenakan biaya?',
  'Bagaimana cara mengetahui status surat yang saya ajukan?',
  'Bagaimana cara mengambil surat yang sudah selesai?',
  'Kapan jam pelayanan kantor desa?',
  'Di mana lokasi kantor Desa Sabah Balau?',
  'Bagaimana cara menghubungi kantor desa?',
  'Apakah tempat tinggal saya termasuk wilayah Desa Sabah Balau?',
  'Bagaimana melaporkan jalan rusak, lampu jalan mati, atau saluran air tersumbat?',
  'Bagaimana informasi bantuan sosial dari pemerintah?',
  'Apakah ada informasi Posyandu dan kegiatan kesehatan?',
  'Kalau status surat sudah Selesai, bagaimana cara menerima atau mengambilnya?',
  'Apakah surat yang diajukan lewat bot dikirim digital atau harus diambil?',
  'Apa arti status Selesai pada permohonan surat?',
  'Berapa lama sampai surat selesai diproses?',
  'Bisakah pengajuan surat dipercepat atau diprioritaskan?',
  'Apakah pengajuan surat lewat bot berbayar?',
  'Bisakah saya mengubah data atau lampiran setelah permohonan dikirim?',
  'Bagaimana membatalkan pengajuan yang sudah mendapat kode?',
  'Apa yang harus saya lakukan jika statusnya Perlu Perbaikan?',
  'Apa yang harus dilakukan jika permohonan surat ditolak?',
  'Bolehkah keluarga mengambil surat atau mengurus permohonan sebagai wakil?',
  'Apakah saya bisa mengecek status permohonan orang lain?',
  'Bolehkah membagikan kode permohonan di grup atau ke orang lain?',
  'Apakah aman mengirim foto persyaratan di chat ini?',
  'Apakah bot bisa memberi tahu status atau keputusan surat dari nama saya?',
  'Saya butuh layanan desa selain jenis surat di menu, bagaimana?',
  'Bagaimana mengetahui apakah alamat saya termasuk wilayah desa ini?',
  'Di mana saya melihat informasi resmi tentang program atau kegiatan desa?',
  '[KONFIRMASI DESA] Siapa perangkat desa dan apa tugas masing-masing?',
  '[KONFIRMASI DESA] Apa fasilitas umum, program, dan kegiatan warga yang tersedia?'
) OR pertanyaan IN (SELECT pertanyaan FROM faq_seed);

INSERT INTO faq (pertanyaan, kata_kunci, jawaban, kategori, aktif)
SELECT pertanyaan, kata_kunci, jawaban, kategori, aktif
FROM faq_seed
ORDER BY n;

COMMIT;
