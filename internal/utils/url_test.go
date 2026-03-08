package url

import (
	"testing"
)

func TestExtractNodeToken(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid default URL",
			rawURL:   "https://my.feishu.cn/wiki/SrR9whoxWiIcuuknF9Rc0TyWnxh",
			expected: "SrR9whoxWiIcuuknF9Rc0TyWnxh",
			wantErr:  false,
		},
		{
			name:     "Valid URL with node param",
			rawURL:   "https://my.feishu.cn/wiki/wikcnXyz123?node=SrR9whoxWiIcuuknF9Rc0TyWnxh",
			expected: "SrR9whoxWiIcuuknF9Rc0TyWnxh",
			wantErr:  false,
		},
		{
			name:     "Invalid structure",
			rawURL:   "https://my.feishu.cn/docx/SrR9whoxWiIcuuknF9Rc0TyWnxh",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Empty URL",
			rawURL:   "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractNodeToken(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractNodeToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ExtractNodeToken() got = %v, want %v", got, tt.expected)
			}
		})
	}
}
