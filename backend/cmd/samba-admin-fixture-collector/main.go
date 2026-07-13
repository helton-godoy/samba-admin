package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters/freebsd"
	"github.com/hu-ufcat/samba-admin-backend/internal/fixturecollector"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "falha na coleta: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("samba-admin-fixture-collector", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "-", "arquivo JSON novo ou - para stdout")
	timeout := flags.Duration("timeout", 15*time.Second, "timeout por comando somente leitura")
	sanitizeHostname := flags.Bool("sanitize-hostname", true, "remove hostname e FQDN")
	sanitizeIPs := flags.Bool("sanitize-ips", true, "remove endereços IP")
	sanitizeDomains := flags.Bool("sanitize-domains", true, "remove nomes DNS")
	sanitizeUsers := flags.Bool("sanitize-users", true, "remove identidades de usuário/grupo")
	sanitizeSIDs := flags.Bool("sanitize-sids", true, "remove SIDs")
	sanitizePaths := flags.Bool("sanitize-paths", true, "remove paths não sistêmicos")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if runtime.GOOS != "freebsd" {
		return fmt.Errorf("coletor recusado em %s: execute no FreeBSD de homologação", runtime.GOOS)
	}
	if *timeout < time.Second || *timeout > time.Minute {
		return errors.New("timeout deve ficar entre 1s e 1m")
	}

	capture := fixturecollector.NewCaptureRunner(freebsd.ExecRunner{MaxOutput: 1 << 20})
	provider := freebsd.New(capture)
	provider.Timeout = *timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	fixture := fixturecollector.Collect(ctx, provider, nil, time.Now())
	fixture.Commands = capture.Commands()
	sanitized, err := fixturecollector.Sanitize(fixture, fixturecollector.SanitizeOptions{
		Hostname: *sanitizeHostname,
		IPs:      *sanitizeIPs,
		Domains:  *sanitizeDomains,
		Users:    *sanitizeUsers,
		SIDs:     *sanitizeSIDs,
		Paths:    *sanitizePaths,
	})
	if err != nil {
		return err
	}

	writer, closeWriter, err := outputWriter(*output, stdout)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(true)
	if err = encoder.Encode(sanitized); err != nil {
		_ = closeWriter()
		return fmt.Errorf("gravar fixture: %w", err)
	}
	if err = closeWriter(); err != nil {
		return fmt.Errorf("fechar fixture: %w", err)
	}
	return nil
}

func outputWriter(path string, stdout io.Writer) (io.Writer, func() error, error) {
	if path == "-" {
		return stdout, func() error { return nil }, nil
	}
	if strings.TrimSpace(path) == "" || !strings.HasSuffix(strings.ToLower(path), ".json") {
		return nil, nil, errors.New("output deve ser um arquivo .json explícito")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("criar output sem sobrescrever: %w", err)
	}
	return file, func() error {
		if err := file.Sync(); err != nil {
			_ = file.Close()
			return err
		}
		return file.Close()
	}, nil
}
