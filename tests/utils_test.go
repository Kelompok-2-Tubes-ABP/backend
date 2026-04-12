package tests

import (
	"testing"

	"financeapi/essentials/utils"
)

// === KUMPULAN TEST UNTUK NLP (Natural Language Processing) ===

// 1. Test deteksi dan konversi teks rupiah ke angka
func TestParseIndonesianAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1 juta", 1000000},
		{"1.5 juta", 1500000},
		{"500 ribu", 500000},
		{"100rb", 100000},
		{"makan siang 50rb dong", 50000},
		{"gaji 10 jt", 10000000},
		{"1000000", 1000000},
		{"no numbers here", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := utils.ParseIndonesianAmount(tt.input)
			if result != tt.expected {
				t.Errorf("ParseIndonesianAmount(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// 2. Test deteksi kategori pengeluaran dari chat natural
func TestExtractExpenseCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"baru selesai makan di mall", "food"},
		{"bayar ojek online", "transport"},
		{"beli baju baru", "shopping"},
		{"nonton bioskop", "entertainment"},
		{"bayar tagihan listrik", "bills"},
		{"beli obat di apotek", "health"},
		{"bayar uang kuliah", "education"},
		{"random pengeluaran", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := utils.ExtractExpenseCategory(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractExpenseCategory(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// 3. Test Ekstraksi Tenor Bulan/Tahun untuk pinjaman/Recurring
func TestExtractTenor(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"cicilan 12 bulan", 12},
		{"tenor 24 bln", 24},
		{"cicilan 2 tahun", 24}, // 2 * 12
		{"pinjaman 3 thn", 36},
		{"tanpa tenor", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := utils.ExtractTenor(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractTenor(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// 4. Test Ekstraksi Suku Bunga Persen via Regex
func TestExtractInterestRate(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"bunga 5%", 5.0},
		{"bunga 1.5 %", 1.5},
		{"bunga 10 persen", 10.0},
		{"tanpa bunga", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := utils.ExtractInterestRate(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractInterestRate(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// === KUMPULAN TEST UNTUK SECURITY/VALIDASI ===

// 5. Test Keamanan Hash Password dan Pencocokan
func TestPasswordHashing(t *testing.T) {
	password := "SecretP@ssw0rd!"

	// Tes Hashing
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if hash == "" || hash == password {
		t.Fatal("Expected strong hash, returned raw or empty")
	}

	// Tes Kecocokan Login Asli
	isValid := utils.CheckPassword(password, hash)
	if !isValid {
		t.Error("Expected password check to return true for correct password")
	}

	// Tes Kecocokan Password Salah (Anti-Bypass)
	isInvalid := utils.CheckPassword("wrongpassword", hash)
	if isInvalid {
		t.Error("Sekuriti Bocor: Password salah bisa divalidasi sebagai benar!")
	}
}

// 6. Test Validasi Email Mahasiswa / Profesional (Regex Validator)
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"mahasiswa.itb+2023@domain.ac.id", true},
		{"invalid-email", false},
		{"@missingusername.com", false},
		{"missingdomain@", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := utils.ValidateEmail(tt.email)
			if result != tt.expected {
				t.Errorf("ValidateEmail(%s) = %v; want %v", tt.email, result, tt.expected)
			}
		})
	}
}

// 7. Test Filter Validasi Input Username dari Karakter Berbahaya
func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		valid    bool
	}{
		{"valid_user", true},
		{"user123_4", true},
		{"ab", false},                // too short
		{"invalid user!", false},     // contains characters not allowed (spasi & seru)
		{"drop_table_users;", false}, // injeksi sederhana
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			valid, _ := utils.ValidateUsername(tt.username)
			if valid != tt.valid {
				t.Errorf("ValidateUsername(%s) = %v; want %v", tt.username, valid, tt.valid)
			}
		})
	}
}
