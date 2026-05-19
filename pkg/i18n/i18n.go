// Package i18n provides internationalization support by loading INI-formatted
// locale files and looking up translations keyed by "section.key".
//
// It is a drop-in replacement for github.com/beego/i18n, exposing the same
// public API that the mindoc project depends on: Tr, IsExist, SetMessage,
// and ReloadLangs.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/ini.v1"
)

// locale stores the parsed INI file for a single language.
type locale struct {
	file *ini.File
	path string
}

var (
	mu       sync.RWMutex
	locales  = make(map[string]*locale)
)

// SetMessage loads an INI translation file for the given language.
// If the language already exists it is replaced.
func SetMessage(lang, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read locale file %q: %w", filePath, err)
	}

	cfg, err := ini.Load(data)
	if err != nil {
		return fmt.Errorf("failed to parse locale file %q: %w", filePath, err)
	}

	mu.Lock()
	locales[lang] = &locale{file: cfg, path: filePath}
	mu.Unlock()

	return nil
}

// IsExist returns true if a locale has been loaded for the given language.
func IsExist(lang string) bool {
	mu.RLock()
	_, ok := locales[lang]
	mu.RUnlock()
	return ok
}

// Tr translates a key for the given language.
//
// The key format is "section.key" — the value before the first dot is treated
// as the INI section name and the remainder as the key within that section.
// If the key is not found, the key itself is returned.
//
// The variadic args parameter is accepted for API compatibility with
// beego/i18n but is currently unused.
func Tr(lang, key string, args ...interface{}) string {
	mu.RLock()
	loc, ok := locales[lang]
	mu.RUnlock()

	if !ok || loc == nil || loc.file == nil {
		return key
	}

	section, item := splitKey(key)
	if section == "" || item == "" {
		return key
	}

	sec, err := loc.file.GetSection(section)
	if err != nil {
		return key
	}

	val := sec.Key(item).String()
	if val == "" {
		return key
	}
	return val
}

// ReloadLangs reloads all previously loaded locale files from disk.
func ReloadLangs(langs ...string) error {
	mu.RLock()
	paths := make(map[string]string)
	for lang, loc := range locales {
		paths[lang] = loc.path
	}
	mu.RUnlock()

	var firstErr error
	for lang, path := range paths {
		if err := SetMessage(lang, path); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// splitKey splits "section.key" at the first dot.
func splitKey(key string) (string, string) {
	idx := strings.Index(key, ".")
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}
