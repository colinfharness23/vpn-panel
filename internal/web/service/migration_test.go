package service

import (
	"net"
	"strings"
	"testing"
)

func TestValidateMigrationRequest(t *testing.T) {
	tests := []struct {
		name    string
		request ServerMigrationRequest
		wantErr bool
	}{
		{name: "valid IPv4 with default port", request: ServerMigrationRequest{Host: "203.0.113.10", Username: "root", Password: "secret"}},
		{name: "valid IPv6", request: ServerMigrationRequest{Host: "2001:db8::10", Port: 2222, Username: "ubuntu", Password: "secret"}},
		{name: "reject loopback", request: ServerMigrationRequest{Host: "127.0.0.1", Port: 22, Username: "root", Password: "secret"}, wantErr: true},
		{name: "reject invalid port", request: ServerMigrationRequest{Host: "203.0.113.10", Port: 70000, Username: "root", Password: "secret"}, wantErr: true},
		{name: "reject unsafe username", request: ServerMigrationRequest{Host: "203.0.113.10", Port: 22, Username: "root;shutdown", Password: "secret"}, wantErr: true},
		{name: "reject empty password", request: ServerMigrationRequest{Host: "203.0.113.10", Port: 22, Username: "root"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateMigrationRequest(test.request)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateMigrationRequest() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestParseMigrationRemoteInfo(t *testing.T) {
	info, err := parseMigrationRemoteInfo("OS_ID=ubuntu\nOS_VERSION=24.04\nARCH=x86_64\nMACHINE_ID=abc\nDISK_FREE_KB=4194304\nEXISTING_INSTALL=1\n")
	if err != nil {
		t.Fatal(err)
	}
	if info.OSID != "ubuntu" || info.OSVersion != "24.04" || info.Arch != "x86_64" || !info.ExistingInstall {
		t.Fatalf("unexpected remote info: %#v", info)
	}
}

func TestMigrationInstallCommandSeparatesChmodAndInstaller(t *testing.T) {
	command := migrationInstallCommand(migrationDeployConfig{
		Repository: "owner/repo", ReleaseTag: "v1.2.3", Domain: "example.com",
		AdminPath: "583104927618350492", AdminUsername: "owner", BootstrapAdminPassword: "Aa1-bootstrap-secret",
		PanelPort: "2053", WebBasePath: "/", DBName: "nova", DBUser: "nova",
		DeferTLS: true, ExpectedPublicIP: "203.0.113.10", PreviousDNSIPs: "198.51.100.20",
	}, "/tmp/install.sh")
	if !strings.Contains(command, "chmod 700 '/tmp/install.sh'\nenv ") {
		t.Fatalf("installer command must start on a new shell command: %q", command)
	}
	if !strings.Contains(command, "NOVA_GITHUB_REPO='owner/repo'") ||
		!strings.Contains(command, "NOVA_ADMIN_PATH='583104927618350492'") ||
		!strings.Contains(command, "NOVA_ADMIN_USERNAME='owner'") ||
		!strings.Contains(command, "NOVA_ADMIN_PASSWORD='Aa1-bootstrap-secret'") ||
		!strings.Contains(command, "NOVA_DEFER_TLS='true'") ||
		!strings.Contains(command, "NOVA_EXPECTED_PUBLIC_IP='203.0.113.10'") ||
		!strings.Contains(command, "NOVA_PREVIOUS_DNS_IPS='198.51.100.20'") ||
		strings.Contains(command, "NOVA_PANEL_PORT=") ||
		!strings.HasSuffix(command, "bash '/tmp/install.sh'") {
		t.Fatalf("installer environment is incomplete: %q", command)
	}
}

func TestMigrationRestoreKeepsTargetSubscriptionSettings(t *testing.T) {
	for _, postgresSource := range []bool{false, true} {
		command := migrationRestoreCommand("/tmp/source.dump", postgresSource)
		for _, required := range []string{
			"${NOVA_SUB_PORT}", "${NOVA_SUB_PATH}", "${NOVA_SUB_JSON_PATH}", "${NOVA_SUB_CLASH_PATH}",
			"https://${NOVA_DOMAIN}", "('webDomain','${NOVA_DOMAIN}')", "('subTitle','NOVA')",
			"('site.url','https://${NOVA_DOMAIN}'", "('site.force_https','true'",
		} {
			if !strings.Contains(command, required) {
				t.Fatalf("restore command does not preserve target setting %s: %q", required, command)
			}
		}
	}
}

func TestMigrationResultURLs(t *testing.T) {
	portal, panel := migrationResultURLs(migrationDeployConfig{Domain: "vpn.example.com", WebBasePath: "/", AdminPath: "583104927618350492"}, "203.0.113.10")
	if portal != "https://vpn.example.com/" || panel != "https://vpn.example.com/583104927618350492/" {
		t.Fatalf("unexpected URLs: %s %s", portal, panel)
	}
}

func TestMigrationDNSRequiresEveryRecordToPointAtTarget(t *testing.T) {
	target := net.ParseIP("203.0.113.10")
	ready, _ := migrationDNSAddressesReady([]net.IPAddr{{IP: target}}, target)
	if !ready {
		t.Fatal("single target DNS record was not ready")
	}
	ready, _ = migrationDNSAddressesReady([]net.IPAddr{{IP: target}, {IP: net.ParseIP("198.51.100.20")}}, target)
	if ready {
		t.Fatal("mixed old/new DNS records were accepted")
	}
}

func TestGenerateMigrationBootstrapPasswordMeetsInstallerPolicy(t *testing.T) {
	password, err := generateMigrationBootstrapPassword()
	if err != nil {
		t.Fatal(err)
	}
	if len(password) < 12 || !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") ||
		!strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") {
		t.Fatalf("generated password does not meet installer policy: %q", password)
	}
}

func TestMigrationErrorTailRedactsCredentials(t *testing.T) {
	output := "normal line\n管理员密码: secret\nNOVA_DB_PASSWORD=another-secret\nfinal line"
	tail := migrationErrorTail(output)
	if strings.Contains(tail, "secret") || strings.Contains(tail, "another-secret") {
		t.Fatalf("migration output leaked credentials: %q", tail)
	}
	if !strings.Contains(tail, "[敏感凭据已隐藏]") {
		t.Fatalf("expected redaction marker: %q", tail)
	}
}
