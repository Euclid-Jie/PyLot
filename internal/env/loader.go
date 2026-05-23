package env

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadGlobalEnv(envFilePath string) (map[string]string, error) {
	if envFilePath == "" {
		return map[string]string{}, nil
	}
	m, err := godotenv.Read(envFilePath)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func MergeEnv(globalEnv, privateEnv map[string]string) []string {
	merged := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				merged[e[:i]] = e[i+1:]
				break
			}
		}
	}
	for k, v := range globalEnv {
		merged[k] = v
	}
	for k, v := range privateEnv {
		merged[k] = v
	}
	result := make([]string, 0, len(merged))
	for k, v := range merged {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func BuildEnvSnapshot(globalEnv, privateEnv map[string]string) string {
	snapshot := make(map[string]string)
	for k, v := range globalEnv {
		snapshot[k] = v
	}
	for k, v := range privateEnv {
		snapshot[k] = v
	}
	b, _ := json.Marshal(snapshot)
	return string(b)
}
