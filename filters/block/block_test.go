package block

import (
	"net/http"
	"strings"
	"testing"
)

type nonBlockingReader struct {
	initialContent []byte
}

func (r *nonBlockingReader) Read(p []byte) (int, error) {
	n := copy(p, r.initialContent)
	r.initialContent = r.initialContent[n:]
	return n, nil
}

func (r *nonBlockingReader) Close() error {
	return nil
}

func TestBlock(t *testing.T) {
	for _, tt := range []struct {
		name    string
		content string
		err     error
	}{
		{
			name:    "small string",
			content: "fox",
			err:     nil,
		},
		{
			name:    "small string",
			content: "foxi",
			err:     nil,
		},
		{
			name:    "small string with match",
			content: "fox.class.foo.blah",
			err:     ErrBlocked,
		},
		{
			name:    "long string",
			content: strings.Repeat("A", 8192),
			err:     nil,
		}} {
		t.Run(tt.name, func(t *testing.T) {
			r := &nonBlockingReader{initialContent: []byte(tt.content)}
			bmb := newBlockBuffer(r, []string{".class"})
			p := make([]byte, len(r.initialContent))
			n, err := bmb.Read(p)
			if err != nil {
				if err == ErrBlocked {
					t.Logf("Stop! Request has some blocked content!")
				} else {
					t.Errorf("Failed to read: %v", err)
				}
			} else if n != len(tt.content) {
				t.Errorf("Failed to read content length %d, got %d", len(tt.content), n)
			}

		})
	}

}

func BenchmarkBlock(b *testing.B) {

	fake := func(source string, len int) string {
		return strings.Repeat(source[:2], len/2) // partially matches target
	}

	for _, tt := range []struct {
		name    string
		tomatch string
		bm      []byte
	}{
		{
			name:    "Small",
			tomatch: ".class",
			bm:      []byte(fake(".class", 10)),
		},
		{
			name:    "Medium",
			tomatch: ".class",
			bm:      []byte(fake(".class", 1000)),
		},
		{
			name:    "Large",
			tomatch: ".class",
			bm:      []byte(fake(".class", 10000)),
		}} {
		b.Run(tt.name, func(b *testing.B) {
			target := &nonBlockingReader{initialContent: tt.bm}
			r := &http.Request{
				Body: target,
			}

			bmb := newBlockBuffer(r.Body, []string{tt.tomatch})
			p := make([]byte, len(target.initialContent))
			for n := 0; n < b.N; n++ {
				bmb.Read(p)
			}
		})
	}
}
