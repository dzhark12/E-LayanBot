package handler

import (
	"context"
	"strings"
	"time"

	"elayan.local/desa/internal/messenger"
)

// phoneKey menormalkan JID nomor telepon seperti yang disimpan pada
// permohonan.wa_jid (tanpa sufiks perangkat).
func phoneKey(jid string) string {
	local := strings.TrimSuffix(jid, "@s.whatsapp.net")
	if device, _, ok := strings.Cut(local, ":"); ok {
		local = device
	}
	return local + "@s.whatsapp.net"
}

// rememberChat menyimpan alamat chat (JID yang dipakai bot untuk membalas, bisa
// berupa @lid) pada permohonan milik nomor tersebut. Notifikasi status dikirim ke
// alamat ini karena terbukti berhasil dipakai untuk membalas. Bersifat best-effort
// dan tidak pernah mengganggu alur percakapan.
func (r *Router) rememberChat(ctx context.Context, message messenger.Message) {
	if !strings.HasSuffix(message.JID, "@lid") && !strings.HasSuffix(message.JID, "@s.whatsapp.net") {
		return
	}
	if message.PhoneJID == "" {
		return
	}
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := r.db.ExecContext(dbCtx, `UPDATE permohonan SET wa_chat_jid=$2 WHERE wa_jid=$1 AND wa_chat_jid IS DISTINCT FROM $2`, phoneKey(message.PhoneJID), message.JID); err != nil {
		r.logger.Printf("chat address could not be saved: %v", err)
	}
}
