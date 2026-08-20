package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

const supabaseLogoBucket = "logos"
const supabaseAvatarBucket = "avatars"

type SupabaseStorage struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

func NewSupabaseStorage(baseURL, serviceKey string) *SupabaseStorage {
	return &SupabaseStorage{
		baseURL:    strings.TrimRight(baseURL, "/"),
		serviceKey: serviceKey,
		httpClient: http.DefaultClient,
	}
}

// UploadLogoToSupabase envia o arquivo para o bucket público 'logos' do Supabase
// e retorna a URL pública final da imagem.
func UploadLogoToSupabase(ctx context.Context, fileReader io.Reader, fileName string) (string, error) {
	baseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if baseURL == "" || serviceKey == "" {
		return "", fmt.Errorf("SUPABASE_URL e SUPABASE_SERVICE_ROLE_KEY devem estar configuradas")
	}

	storage := NewSupabaseStorage(baseURL, serviceKey)
	return storage.UploadLogo(ctx, fileReader, fileName)
}

// UploadAvatarToSupabase envia avatar para o bucket público 'avatars' (path namespaced por tenant).
// objectPath típico: "{estabelecimento_id}/{profissional_id}-{nanos}.png"
func UploadAvatarToSupabase(ctx context.Context, fileReader io.Reader, objectPath string) (string, error) {
	baseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if baseURL == "" || serviceKey == "" {
		return "", fmt.Errorf("SUPABASE_URL e SUPABASE_SERVICE_ROLE_KEY devem estar configuradas")
	}

	storage := NewSupabaseStorage(baseURL, serviceKey)
	return storage.UploadAvatar(ctx, fileReader, objectPath)
}

func (s *SupabaseStorage) UploadLogo(ctx context.Context, fileReader io.Reader, fileName string) (string, error) {
	safeName := path.Base(strings.TrimSpace(fileName))
	if safeName == "" || safeName == "." {
		return "", fmt.Errorf("nome de arquivo inválido")
	}
	return s.uploadPublicObject(ctx, supabaseLogoBucket, safeName, fileReader, true)
}

func (s *SupabaseStorage) UploadAvatar(ctx context.Context, fileReader io.Reader, objectPath string) (string, error) {
	return s.uploadPublicObject(ctx, supabaseAvatarBucket, objectPath, fileReader, false)
}

func (s *SupabaseStorage) uploadPublicObject(
	ctx context.Context,
	bucket, objectPath string,
	fileReader io.Reader,
	upsert bool,
) (string, error) {
	safePath := strings.Trim(strings.ReplaceAll(objectPath, "\\", "/"), "/")
	if safePath == "" || strings.Contains(safePath, "..") {
		return "", fmt.Errorf("caminho de objeto inválido")
	}

	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, bucket, safePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, fileReader)
	if err != nil {
		return "", fmt.Errorf("criar requisição de upload: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", contentTypeFromExt(safePath))
	if upsert {
		req.Header.Set("x-upsert", "true")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("enviar arquivo ao Supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("upload rejeitado pelo Supabase (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, bucket, safePath)
	return publicURL, nil
}

func contentTypeFromExt(fileName string) string {
	switch strings.ToLower(path.Ext(fileName)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
