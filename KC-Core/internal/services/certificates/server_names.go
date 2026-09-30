package certificates

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

// serverCertificateNames are the names a server certificate carries: its
// subject and the further names stations reach the service center by.
type serverCertificateNames struct {
	subject  string
	altNames []string
}

// all lists the subject, then the alternative names.
func (n serverCertificateNames) all() []string {
	return append([]string{n.subject}, n.altNames...)
}

// plannedServerNames are the names the next server certificate carries. The
// host of the configured external URL names it, else the current
// certificate's subject; the current certificate's names and the configured
// names stay on it, so renewal never drops a name stations connect by.
func (s *Service) plannedServerNames(ctx context.Context) serverCertificateNames {
	current := s.currentServerNames(ctx)
	subject := config.ExternalBSSCIHost(s.settings.protocol)
	if subject == "" {
		subject = current.subject
	}
	if subject == "" {
		subject = s.fallbackHost()
	}
	return serverCertificateNames{
		subject:  subject,
		altNames: distinctNames(subject, append(append([]string{}, s.settings.serverNames...), current.all()...)),
	}
}

// currentServerNames reads the names of the certificate in the certificates
// directory; none when there is no readable certificate.
func (s *Service) currentServerNames(ctx context.Context) serverCertificateNames {
	cert := s.readCertificate(ctx, filepath.Join(s.settings.certsDir, serverCertFileName))
	if cert == nil {
		return serverCertificateNames{}
	}
	names := append([]string{}, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		names = append(names, ip.String())
	}
	return serverCertificateNames{subject: cert.Subject.CommonName, altNames: names}
}

// fallbackHost names a first server certificate when nothing else does.
func (s *Service) fallbackHost() string {
	if !config.IsWildcardHost(s.settings.protocol.BSCIHost) {
		return s.settings.protocol.BSCIHost
	}
	return config.DefaultCertificatesHostname
}

// distinctNames keeps each non-blank name once, case-insensitively, leaving
// out the subject.
func distinctNames(subject string, names []string) []string {
	seen := map[string]bool{strings.ToLower(subject): true}
	var out []string
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(name))
	}
	return out
}
