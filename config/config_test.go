package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Mock(t *testing.T) {
	cfg, err := LoadConfig("", "", true)
	if err != nil {
		t.Fatalf("LoadConfig with mock=true should succeed: %v", err)
	}
	if cfg.Provider != ProviderMock {
		t.Errorf("expected provider %q, got %q", ProviderMock, cfg.Provider)
	}
	if cfg.Model != DefaultModelMock {
		t.Errorf("expected model %q, got %q", DefaultModelMock, cfg.Model)
	}
	if !cfg.Mock {
		t.Errorf("expected Mock=true")
	}
}

func TestLoadConfig_OverrideWithKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key-gemini")
	cfg, err := LoadConfig(ProviderGemini, "", false)
	if err != nil {
		t.Fatalf("LoadConfig with override+key should succeed: %v", err)
	}
	if cfg.Provider != ProviderGemini {
		t.Errorf("expected provider %q, got %q", ProviderGemini, cfg.Provider)
	}
	if cfg.APIKey != "test-key-gemini" {
		t.Errorf("expected API key %q, got %q", "test-key-gemini", cfg.APIKey)
	}
	if cfg.Model != DefaultModelGemini {
		t.Errorf("expected default model %q, got %q", DefaultModelGemini, cfg.Model)
	}
}

func TestLoadConfig_OverrideNoKey(t *testing.T) {
	// Ensure no env var is set
	t.Setenv("GEMINI_API_KEY", "")
	_, err := LoadConfig(ProviderGemini, "", false)
	if err == nil {
		t.Fatal("LoadConfig with override but no key should fail")
	}
}

func TestLoadConfig_AutoDetectOrder(t *testing.T) {
	// Set multiple keys — Gemini has priority
	t.Setenv("OPENAI_API_KEY", "key-openai")
	t.Setenv("GEMINI_API_KEY", "key-gemini")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect should succeed: %v", err)
	}
	if cfg.Provider != ProviderGemini {
		t.Errorf("expected Gemini to have priority, got %q", cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "key-openai")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect OpenAI should succeed: %v", err)
	}
	if cfg.Provider != ProviderOpenAI {
		t.Errorf("expected provider %q, got %q", ProviderOpenAI, cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "key-anthropic")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect Anthropic should succeed: %v", err)
	}
	if cfg.Provider != ProviderAnthropic {
		t.Errorf("expected provider %q, got %q", ProviderAnthropic, cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectOpenRouter(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "key-openrouter")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect OpenRouter should succeed: %v", err)
	}
	if cfg.Provider != ProviderOpenRouter {
		t.Errorf("expected provider %q, got %q", ProviderOpenRouter, cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectNoKeys(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	_, err := LoadConfig("", "", false)
	if err == nil {
		t.Fatal("LoadConfig with no keys should fail")
	}
}

func TestLoadConfig_ModelOverride(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	cfg, err := LoadConfig(ProviderGemini, "custom-model", false)
	if err != nil {
		t.Fatalf("LoadConfig with model override should succeed: %v", err)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("expected model %q, got %q", "custom-model", cfg.Model)
	}
}

func TestAutoDetectProvider(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gk")
	t.Setenv("OPENAI_API_KEY", "ok")
	prov, key := autoDetectProvider()
	if prov != ProviderGemini {
		t.Errorf("expected Gemini priority, got %q", prov)
	}
	if key != "gk" {
		t.Errorf("expected key 'gk', got %q", key)
	}
}

func TestAutoDetectProvider_None(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	prov, key := autoDetectProvider()
	if prov != "" || key != "" {
		t.Errorf("expected empty provider and key, got %q / %q", prov, key)
	}
}

func TestGetAPIKeyForProvider(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gk")
	t.Setenv("OPENAI_API_KEY", "ok")
	t.Setenv("ANTHROPIC_API_KEY", "ak")
	t.Setenv("OPENROUTER_API_KEY", "rk")

	tests := []struct {
		provider string
		want     string
	}{
		{ProviderGemini, "gk"},
		{ProviderOpenAI, "ok"},
		{ProviderAnthropic, "ak"},
		{ProviderOpenRouter, "rk"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := getAPIKeyForProvider(tt.provider)
		if got != tt.want {
			t.Errorf("getAPIKeyForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestGetDefaultModel(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{ProviderGemini, DefaultModelGemini},
		{ProviderOpenAI, DefaultModelOpenAI},
		{ProviderAnthropic, DefaultModelAnthropic},
		{ProviderOpenRouter, DefaultModelOpenRouter},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := getDefaultModel(tt.provider)
		if got != tt.want {
			t.Errorf("getDefaultModel(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestLoadStandards_Embedded(t *testing.T) {
	standards, err := LoadStandards()
	if err != nil {
		t.Fatalf("LoadStandards embedded should succeed: %v", err)
	}
	if len(standards) == 0 {
		t.Fatal("expected at least one standard from embedded defaults")
	}
}

func TestLoadStandards_Override(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	overrideContent := []byte(`standards:
  - id: custom-test
    name: Custom Test Standard
    description: A test standard
    target_files: ["01_domain_model_use_cases.md"]
    criteria: "Must pass"
    min_score: 80`)
	if err := os.WriteFile("standards.yaml", overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	standards, err := LoadStandards()
	if err != nil {
		t.Fatalf("LoadStandards with override should succeed: %v", err)
	}
	if len(standards) != 1 || standards[0].ID != "custom-test" {
		t.Errorf("expected 1 custom standard, got %d (first=%q)", len(standards), standards[0].ID)
	}
}

func TestLoadTemplates_Embedded(t *testing.T) {
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates embedded should succeed: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("expected at least one template from embedded defaults")
	}
}

func TestLoadTemplates_Override(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	overrideContent := []byte(`templates:
  - file_name: "test.md"
    name: "Test Template"
    prompt: "Generate a test document"`)
	if err := os.WriteFile("templates.yaml", overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates with override should succeed: %v", err)
	}
	if len(templates) != 1 || templates[0].FileName != "test.md" {
		t.Errorf("expected 1 custom template, got %d", len(templates))
	}
}

func TestLoadBlueprints_Override(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	overrideContent := []byte(`blueprints:
  - id: custom-bp
    name: Custom Blueprint
    description: A test blueprint
    facts:
      functional: "Custom functional"
      structural: "Custom structural"
      security: "Custom security"
      compliance: "Custom compliance"`)
	if err := os.WriteFile("blueprints.yaml", overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	blueprints, err := LoadBlueprints()
	if err != nil {
		t.Fatalf("LoadBlueprints with override should succeed: %v", err)
	}
	if len(blueprints) != 1 || blueprints[0].ID != "custom-bp" {
		t.Errorf("expected 1 custom blueprint, got %d", len(blueprints))
	}
}

func TestLoadSettings_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Ensure no .synthspec dir exists
	os.RemoveAll(".synthspec")

	// Override home dir to prevent global settings from interfering
	t.Setenv("HOME", tmpDir)
	if _, err := GetGlobalSettingsPath(); err == nil {
		os.RemoveAll(filepath.Join(tmpDir, ".synthspec"))
	}

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings with no files should return defaults: %v", err)
	}
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("expected timeout %d, got %d", DefaultTimeoutSeconds, s.TimeoutSeconds)
	}
	if s.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected max retries %d, got %d", DefaultMaxRetries, s.MaxRetries)
	}
	if s.DefaultOutputFolder != DefaultOutputFolderValue {
		t.Errorf("expected output folder %q, got %q", DefaultOutputFolderValue, s.DefaultOutputFolder)
	}
	if s.Debug {
		t.Errorf("expected Debug=false by default")
	}
}

func TestSaveAndLoadSettings_Local(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmpDir)

	s := &Settings{
		TimeoutSeconds:      60,
		MaxRetries:          3,
		DefaultOutputFolder: "./custom_out",
		Debug:               true,
	}

	if err := SaveSettings(s, false); err != nil {
		t.Fatalf("SaveSettings local failed: %v", err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save failed: %v", err)
	}
	if loaded.TimeoutSeconds != 60 {
		t.Errorf("expected timeout 60, got %d", loaded.TimeoutSeconds)
	}
	if loaded.MaxRetries != 3 {
		t.Errorf("expected retries 3, got %d", loaded.MaxRetries)
	}
	if loaded.DefaultOutputFolder != "./custom_out" {
		t.Errorf("expected folder %q, got %q", "./custom_out", loaded.DefaultOutputFolder)
	}
	if !loaded.Debug {
		t.Errorf("expected Debug=true")
	}
}

func TestLoadSettings_LocalOverridesGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	// Set USERPROFILE (Windows) or HOME (Unix) so global path is distinct from local
	homeDir := filepath.Join(tmpDir, "homeuser")
	os.MkdirAll(homeDir, 0755)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOME", homeDir)

	// Write global settings
	globalDir := filepath.Join(homeDir, ".synthspec")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalData := `{"timeout_seconds": 500, "max_retries": 20, "default_output_folder": "./global_out", "debug": true}`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.json"), []byte(globalData), 0644); err != nil {
		t.Fatal(err)
	}

	// Write local settings (should override global)
	localDir := ".synthspec"
	os.MkdirAll(localDir, 0755)
	localData := `{"timeout_seconds": 100, "max_retries": 5}`
	if err := os.WriteFile(filepath.Join(localDir, "settings.json"), []byte(localData), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}
	if loaded.TimeoutSeconds != 100 {
		t.Errorf("expected local timeout 100, got %d", loaded.TimeoutSeconds)
	}
	if loaded.MaxRetries != 5 {
		t.Errorf("expected local retries 5, got %d", loaded.MaxRetries)
	}
	// Global debug should NOT be overridden since local doesn't have it
	if !loaded.Debug {
		t.Errorf("expected Debug=true from global (not overridden)")
	}
}

func TestMergeSettingsFromFile(t *testing.T) {
	s := &Settings{
		TimeoutSeconds:      DefaultTimeoutSeconds,
		MaxRetries:          DefaultMaxRetries,
		DefaultOutputFolder: DefaultOutputFolderValue,
	}

	// Non-existent file — should no-op
	mergeSettingsFromFile(s, "nonexistent.json")
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("non-existent file should not modify settings")
	}

	// Malformed JSON — should no-op
	tmpFile := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(tmpFile, []byte(`{bad json`), 0644)
	mergeSettingsFromFile(s, tmpFile)
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("malformed JSON should not modify settings")
	}

	// Valid partial file — missing MaxRetries -> JSON zero value 0 -> code's `>= 0` triggers
	validFile := filepath.Join(t.TempDir(), "good.json")
	os.WriteFile(validFile, []byte(`{"timeout_seconds": 99}`), 0644)
	mergeSettingsFromFile(s, validFile)
	if s.TimeoutSeconds != 99 {
		t.Errorf("expected timeout 99, got %d", s.TimeoutSeconds)
	}
	if s.MaxRetries != 0 {
		t.Errorf("expected MaxRetries 0 (overwritten by mergeSettingsFromFile), got %d", s.MaxRetries)
	}
}

func assertBlueprintValid(t *testing.T, bp Blueprint) {
	if bp.Name == "" || bp.Description == "" || bp.Facts.Functional == "" {
		t.Errorf("%s blueprint fields should not be empty", bp.ID)
	}
}

func TestLoadBlueprints(t *testing.T) {
	blueprints, err := LoadBlueprints()
	if err != nil {
		t.Fatalf("failed to load blueprints: %v", err)
	}

	if len(blueprints) < 2 {
		t.Errorf("expected at least 2 default blueprints, got %d", len(blueprints))
	}

	var hasFintech, hasCrud bool
	for _, bp := range blueprints {
		switch bp.ID {
		case "fintech-saas":
			hasFintech = true
			assertBlueprintValid(t, bp)
		case "internal-crud":
			hasCrud = true
			assertBlueprintValid(t, bp)
		}
	}

	if !hasFintech {
		t.Error("missing fintech-saas blueprint")
	}
	if !hasCrud {
		t.Error("missing internal-crud blueprint")
	}
}

func setupTestSettingsEnvironment(_ *testing.T) (string, string, bool, []byte, bool, []byte) {
	localPath := GetLocalSettingsPath()
	origExist := false
	var origData []byte
	if _, err := os.Stat(localPath); err == nil {
		origExist = true
		if data, err := os.ReadFile(localPath); err == nil {
			origData = data
		}
		_ = os.Remove(localPath)
	}

	globalPath, gErr := GetGlobalSettingsPath()
	origGlobalExist := false
	var origGlobalData []byte
	if gErr == nil {
		if _, err := os.Stat(globalPath); err == nil {
			origGlobalExist = true
			if data, err := os.ReadFile(globalPath); err == nil {
				origGlobalData = data
			}
			_ = os.Remove(globalPath)
		}
	}

	return localPath, globalPath, origExist, origData, origGlobalExist, origGlobalData
}

func restoreTestSettingsEnvironment(localPath, globalPath string, origExist bool, origData []byte, origGlobalExist bool, origGlobalData []byte) {
	if origExist {
		_ = os.WriteFile(localPath, origData, 0644)
	} else {
		_ = os.Remove(localPath)
	}
	if origGlobalExist {
		_ = os.WriteFile(globalPath, origGlobalData, 0644)
	}
}

func verifyLoadedSettings(t *testing.T) {
	s2, err := LoadSettings()
	if err != nil {
		t.Fatalf("failed to load modified settings: %v", err)
	}

	if s2.TimeoutSeconds != 120 {
		t.Errorf("expected loaded timeout 120, got %d", s2.TimeoutSeconds)
	}
	if s2.MaxRetries != 5 {
		t.Errorf("expected loaded max retries 5, got %d", s2.MaxRetries)
	}
	if s2.DefaultOutputFolder != "./test_output" {
		t.Errorf("expected loaded default output folder './test_output', got %s", s2.DefaultOutputFolder)
	}
}

func TestSettings(t *testing.T) {
	localPath, globalPath, origExist, origData, origGlobalExist, origGlobalData := setupTestSettingsEnvironment(t)
	defer restoreTestSettingsEnvironment(localPath, globalPath, origExist, origData, origGlobalExist, origGlobalData)

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("expected default timeout %d, got %d", DefaultTimeoutSeconds, s.TimeoutSeconds)
	}

	s.TimeoutSeconds = 120
	s.MaxRetries = 5
	s.DefaultOutputFolder = "./test_output"

	if err := SaveSettings(s, false); err != nil {
		t.Fatalf("failed to save local settings: %v", err)
	}

	verifyLoadedSettings(t)
}
