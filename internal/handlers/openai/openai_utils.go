package openai

// chunkText splits a string into rune-safe chunks of approximately size n.
func chunkText(s string, n int) []string {
	if n <= 0 {
		n = 20
	}
	r := []rune(s)
	if len(r) == 0 {
		return []string{""}
	}
	out := make([]string, 0, (len(r)/n)+1)
	for i := 0; i < len(r); i += n {
		end := i + n
		if end > len(r) {
			end = len(r)
		}
		out = append(out, string(r[i:end]))
	}
	return out
}

// fakeChunkSize returns chunk size for text delta splitting in fake streaming paths.
func (h *Handler) fakeChunkSize() int {
	if h.cfg.FakeStreamingChunkSize > 0 {
		return h.cfg.FakeStreamingChunkSize
	}
	return 20
}
