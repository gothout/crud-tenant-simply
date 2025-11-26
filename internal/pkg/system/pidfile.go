package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func SavePID(path string, pid int) error {
	if path == "" {
		return errors.New("caminho do arquivo PID não informado")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("falha ao criar diretório do PID: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("arquivo PID já existe em %s - o servidor pode estar em execução", path)
	}

	return os.WriteFile(path, []byte(fmt.Sprintf("%d", pid)), 0o644)
}

func LoadPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("falha ao ler arquivo PID: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("PID inválido em %s: %w", path, err)
	}
	return pid, nil
}

func RemovePID(path string) {
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

func TerminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("não foi possível localizar o processo %d: %w", pid, err)
	}

	if runtime.GOOS == "windows" {
		return proc.Kill()
	}
	return proc.Signal(syscall.SIGTERM)
}
