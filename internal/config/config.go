package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type ValidatorConfig struct {
	NamespaceSelector *metav1.LabelSelector `yaml:"namespaceSelector"`
	SubdomainLabel    string                `yaml:"subdomainLabel"`
	MatchDomains      []string              `yaml:"matchDomains"`
}

type ConfigManager struct {
	mu       sync.RWMutex
	config   *ValidatorConfig
	lastHash [32]byte
}

func (cm *ConfigManager) Get() *ValidatorConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)
	cm.mu.RLock()
	sameHash := hash == cm.lastHash
	cm.mu.RUnlock()

	if sameHash {
		return nil
	}

	var cfg ValidatorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal WebhookConfig: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = &cfg
	cm.lastHash = hash
	return nil
}
