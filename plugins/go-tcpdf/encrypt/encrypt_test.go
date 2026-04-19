package encrypt

import (
	"bytes"
	"testing"
)

func newTestEncrypt(mode EncryptionMode) (*Encrypt, error) {
	return New(Config{
		Mode:          mode,
		UserPassword:  "user",
		OwnerPassword: "owner",
		Permissions:   PermPrint | PermCopy,
	})
}

// ---- construction -------------------------------------------------------

func TestNewRC4_40(t *testing.T) {
	enc, err := newTestEncrypt(EncRC4_40)
	if err != nil {
		t.Fatal(err)
	}
	if enc.KeyLength() != 40 {
		t.Fatalf("key length: want 40 got %d", enc.KeyLength())
	}
	if enc.Revision() != 2 {
		t.Fatalf("revision: want 2 got %d", enc.Revision())
	}
	if enc.Version() != 1 {
		t.Fatalf("version: want 1 got %d", enc.Version())
	}
}

func TestNewRC4_128(t *testing.T) {
	enc, err := newTestEncrypt(EncRC4_128)
	if err != nil {
		t.Fatal(err)
	}
	if enc.KeyLength() != 128 {
		t.Fatalf("key length: want 128 got %d", enc.KeyLength())
	}
	if enc.Revision() != 3 {
		t.Fatalf("revision: want 3 got %d", enc.Revision())
	}
}

func TestNewAES128(t *testing.T) {
	enc, err := newTestEncrypt(EncAES_128)
	if err != nil {
		t.Fatal(err)
	}
	if enc.KeyLength() != 128 {
		t.Fatalf("key length: want 128 got %d", enc.KeyLength())
	}
	if enc.Revision() != 4 {
		t.Fatalf("revision: want 4 got %d", enc.Revision())
	}
	if enc.Version() != 4 {
		t.Fatalf("version: want 4 got %d", enc.Version())
	}
}

func TestNewAES256(t *testing.T) {
	enc, err := newTestEncrypt(EncAES_256)
	if err != nil {
		t.Fatal(err)
	}
	if enc.KeyLength() != 256 {
		t.Fatalf("key length: want 256 got %d", enc.KeyLength())
	}
	if enc.Revision() != 6 {
		t.Fatalf("revision: want 6 got %d", enc.Revision())
	}
	if enc.Version() != 5 {
		t.Fatalf("version: want 5 got %d", enc.Version())
	}
}

// ---- disabled -----------------------------------------------------------

func TestDisabledNil(t *testing.T) {
	var enc *Encrypt
	if !enc.Disabled() {
		t.Fatal("nil Encrypt should be disabled")
	}
}

func TestDisabledNone(t *testing.T) {
	enc := &Encrypt{cfg: Config{Mode: EncNone}}
	if !enc.Disabled() {
		t.Fatal("EncNone should be disabled")
	}
}

func TestNotDisabled(t *testing.T) {
	enc, _ := newTestEncrypt(EncRC4_40)
	if enc.Disabled() {
		t.Fatal("should not be disabled")
	}
}

// ---- keys ---------------------------------------------------------------

func TestFileIDNotEmpty(t *testing.T) {
	enc, _ := newTestEncrypt(EncRC4_40)
	if len(enc.FileID()) == 0 {
		t.Fatal("file ID should not be empty")
	}
}

func TestFileIDUnique(t *testing.T) {
	enc1, _ := newTestEncrypt(EncRC4_40)
	enc2, _ := newTestEncrypt(EncRC4_40)
	if bytes.Equal(enc1.FileID(), enc2.FileID()) {
		t.Fatal("file IDs should be unique per instance")
	}
}

func TestUserKeyLength(t *testing.T) {
	for _, mode := range []EncryptionMode{EncRC4_40, EncRC4_128, EncAES_128} {
		enc, err := newTestEncrypt(mode)
		if err != nil {
			t.Fatalf("mode %v: %v", mode, err)
		}
		if len(enc.UserKey()) == 0 {
			t.Fatalf("mode %v: user key empty", mode)
		}
		if len(enc.OwnerKey()) == 0 {
			t.Fatalf("mode %v: owner key empty", mode)
		}
	}
}

func TestAES256ExtraKeys(t *testing.T) {
	enc, err := newTestEncrypt(EncAES_256)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc.UE()) == 0 {
		t.Fatal("AES-256: /UE should not be empty")
	}
	if len(enc.OE()) == 0 {
		t.Fatal("AES-256: /OE should not be empty")
	}
	if len(enc.Perms()) != 16 {
		t.Fatalf("AES-256: /Perms should be 16 bytes, got %d", len(enc.Perms()))
	}
}

// ---- encryption ---------------------------------------------------------

func TestEncryptRC4Passthrough(t *testing.T) {
	enc, _ := newTestEncrypt(EncRC4_40)
	enc.SetObjNum(1, 0)
	plain := []byte("Hello, World!")
	cipher, err := enc.EncryptBytes(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(cipher, plain) {
		t.Fatal("encrypted bytes should differ from plaintext")
	}
}

func TestEncryptRC4Symmetric(t *testing.T) {
	enc, _ := newTestEncrypt(EncRC4_40)
	enc.SetObjNum(5, 0)
	plain := []byte("Secret message")
	cipher, _ := enc.EncryptBytes(plain)
	// RC4 is symmetric: encrypt again to get back plaintext
	enc.SetObjNum(5, 0)
	plain2, err := enc.EncryptBytes(cipher)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, plain2) {
		t.Fatalf("RC4 round-trip failed: got %q", plain2)
	}
}

func TestEncryptRC4_128(t *testing.T) {
	enc, _ := newTestEncrypt(EncRC4_128)
	enc.SetObjNum(1, 0)
	plain := []byte("Test data")
	cipher, err := enc.EncryptBytes(plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(cipher) == 0 {
		t.Fatal("cipher should not be empty")
	}
}

func TestEncryptAES128ProducesIV(t *testing.T) {
	enc, _ := newTestEncrypt(EncAES_128)
	enc.SetObjNum(1, 0)
	plain := []byte("AES test data")
	cipher, err := enc.EncryptBytes(plain)
	if err != nil {
		t.Fatal(err)
	}
	// AES-CBC output = IV(16) + encrypted blocks
	if len(cipher) < 16 {
		t.Fatalf("AES-128 output too short: %d", len(cipher))
	}
}

func TestEncryptStringPassthrough(t *testing.T) {
	var enc *Encrypt // nil = disabled
	out, err := enc.EncryptBytes([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "test" {
		t.Fatal("disabled encrypt should passthrough")
	}
}

func TestEncryptDifferentObjects(t *testing.T) {
	enc, _ := newTestEncrypt(EncRC4_128)
	plain := []byte("same data")
	enc.SetObjNum(1, 0)
	c1, _ := enc.EncryptBytes(plain)
	enc.SetObjNum(2, 0)
	c2, _ := enc.EncryptBytes(plain)
	if bytes.Equal(c1, c2) {
		t.Fatal("different objects should produce different ciphertext")
	}
}

// ---- permissions --------------------------------------------------------

func TestPermConstants(t *testing.T) {
	// Permissions should be combinable
	p := PermPrint | PermCopy | PermAnnot
	if p == 0 {
		t.Fatal("combined permissions should not be zero")
	}
	if p&PermPrint == 0 {
		t.Fatal("PermPrint bit should be set")
	}
	if p&PermModify != 0 {
		t.Fatal("PermModify should not be set")
	}
}

func TestPermAll(t *testing.T) {
	all := PermAll
	required := []int{PermPrint, PermModify, PermCopy, PermAnnot, PermFillForms,
		PermExtract, PermAssemble, PermPrintHighQual}
	for _, p := range required {
		if all&p == 0 {
			t.Fatalf("PermAll missing permission 0x%x", p)
		}
	}
}
