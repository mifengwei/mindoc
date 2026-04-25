package conf

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// configStore 线程安全的配置存储
var (
	configStore   map[string]string
	configMu      sync.RWMutex
	configFile    string
)

// Config 是兼容字段（部分代码可能引用）
var Config interface{}

// InitConfig 初始化配置，支持 Beego 的 ${VAR||default} 语法
func InitConfig(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	configMu.Lock()
	defer configMu.Unlock()

	configFile = file
	configStore = make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// 跳过 section 头
		if strings.HasPrefix(line, "[") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")
		val = expandBeegoEnvVars(val)
		configStore[key] = val
	}

	return scanner.Err()
}

// expandBeegoEnvVars 替换 ${VAR} 和 ${VAR||default} 格式
func expandBeegoEnvVars(s string) string {
	for {
		start := strings.Index(s, "${")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "}")
		if end == -1 {
			break
		}
		end += start

		inner := s[start+2 : end]
		parts := strings.SplitN(inner, "||", 2)
		key := strings.TrimSpace(parts[0])
		var replacement string
		if val, ok := os.LookupEnv(key); ok {
			replacement = val
		} else if len(parts) > 1 {
			replacement = strings.TrimSpace(parts[1])
		}
		s = s[:start] + replacement + s[end+1:]
	}
	return s
}

// GetString 获取字符串配置值
func GetString(key string) (string, error) {
	return GetDefaultString(key, ""), nil
}

// GetDefaultString 获取字符串配置值，带默认值
func GetDefaultString(key string, defaultValue string) string {
	configMu.RLock()
	defer configMu.RUnlock()
	if configStore == nil {
		return defaultValue
	}
	if val, ok := configStore[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

// GetDefaultInt 获取整数配置值，带默认值
func GetDefaultInt(key string, defaultValue int) int {
	s := GetDefaultString(key, "")
	if s == "" {
		return defaultValue
	}
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetDefaultBool 获取布尔配置值，带默认值
func GetDefaultBool(key string, defaultValue bool) bool {
	s := GetDefaultString(key, "")
	if s == "" {
		return defaultValue
	}
	return strings.EqualFold(s, "true") || s == "1"
}

// SetConfigValue 设置配置值
func SetConfigValue(key string, value interface{}) error {
	configMu.Lock()
	defer configMu.Unlock()
	if configStore == nil {
		return nil
	}
	configStore[key] = fmt.Sprintf("%v", value)
	return nil
}

// ReloadConfig 重新加载配置文件
func ReloadConfig() error {
	if configFile == "" {
		return nil
	}
	return InitConfig(configFile)
}
