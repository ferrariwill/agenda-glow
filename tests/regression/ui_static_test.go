//go:build regression

package regression_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUI_ActionsDropdownUsaPortal(t *testing.T) {
	path := filepath.Join(repoRoot(t), "frontend-react", "src", "components", "ui", "ActionsDropdown.tsx")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler ActionsDropdown: %v", err)
	}
	for _, needle := range []string{"createPortal", "fixed z-[200]", "overflow-y-auto"} {
		if !strings.Contains(string(b), needle) {
			t.Fatalf("ActionsDropdown sem %q — regressão de menu cortado", needle)
		}
	}
}

func TestUI_DonaSidebarTemWhatsApp(t *testing.T) {
	path := filepath.Join(repoRoot(t), "frontend-react", "src", "components", "dona", "DonaSidebar.tsx")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler DonaSidebar: %v", err)
	}
	if src := string(b); !strings.Contains(src, "/admin/whatsapp") || !strings.Contains(src, "WhatsApp") {
		t.Fatal("sidebar Dona sem item WhatsApp → /admin/whatsapp")
	}
}

func TestUI_WhatsAppStateHelper(t *testing.T) {
	path := filepath.Join(repoRoot(t), "frontend-react", "src", "utils", "whatsappIntegration.ts")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler whatsappIntegration.ts: %v", err)
	}
	if !strings.Contains(string(b), "state=beleza_") {
		t.Fatal("helper front sem state=beleza_")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
