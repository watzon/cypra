// Package main provides the Cypra CLI entrypoint.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/watzon/cypra/dashboard"
	"github.com/watzon/cypra/internal/auth/invite"
	_ "github.com/watzon/cypra/internal/auth/upstream/providers"
	"github.com/watzon/cypra/internal/bootstrap"
	cypra "github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/db"
	"github.com/watzon/cypra/internal/email"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/logging"
	"github.com/watzon/cypra/internal/migrate"
	"github.com/watzon/cypra/internal/oidc"
	"github.com/watzon/cypra/internal/storage/openstore"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	slog.SetDefault(slog.New(logging.RedactingHandler{Handler: slog.NewTextHandler(os.Stderr, nil)}))
	if err := run(os.Args[1:]); err != nil {
		var exitErr exitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.error)
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return exitError{code: 2, error: "cypra: command required"}
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "migrate":
		return runMigrate()
	case "admin":
		return runAdmin(args[1:])
	case "export":
		return runExport(args[1:])
	case "import":
		return runImport(args[1:])
	case "version":
		return runVersion(args[1:])
	default:
		return exitError{code: 2, error: "cypra: unknown command " + args[0]}
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	skipMigrate := fs.Bool("skip-migrate", false, "skip boot-time migration check")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	publicBaseURL := envDefault("PUBLIC_BASE_URL", "https://cypra.localhost")
	if err := validatePublicBaseURL(publicBaseURL); err != nil {
		return err
	}
	dbConn, err := openDB(envDefault("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	if !*skipMigrate {
		pending, err := migrate.Pending(context.Background(), dbConn, "")
		if err != nil {
			return err
		}
		if pending {
			return exitError{code: 4, error: "Pending migrations. Run `cypra migrate`."}
		}
	}
	gormDB, err := gorm.Open(postgres.Open(envDefault("DATABASE_URL", "")), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open gorm db: %w", err)
	}
	if err := gormDB.Use(db.TenantPlugin{}); err != nil {
		return fmt.Errorf("install tenant db plugin: %w", err)
	}
	devMode := os.Getenv("LOG_LEVEL") == "debug"
	// Trust X-Cypra-* elevation headers only when explicitly opted in. Defaults
	// off so a misbehaving proxy (or a stray header from the dashboard) can
	// never escalate the actor.
	trustDevHeaders := os.Getenv("CYPRA_TRUST_DEV_HEADERS") == "1"
	masterKey, err := loadMasterKey()
	if err != nil {
		return err
	}
	store, storeKind, err := openstore.New(openstore.LoadConfigFromEnv(masterKey))
	if err != nil {
		return err
	}
	blockTerminalEmail := terminalEmailBlocked(publicBaseURL)
	server, err := httpserver.New(httpserver.Options{DB: dbConn, TenantDB: db.NewTenantScopedDB(gormDB), PublicBaseURL: publicBaseURL, Version: version, Commit: commit, DevOpenAPI: devMode, KEKLoaded: masterKeyConfigured(), MasterKey: masterKey, GoogleAuthURL: os.Getenv("CYPRA_GOOGLE_AUTH_URL"), DashboardFS: dashboard.Files, DashboardDev: envDefault("VITE_DEV_SERVER", "http://127.0.0.1:5173"), Storage: store, StorageBackend: storeKind, TrustDevHeaders: trustDevHeaders, TrustedProxyHeaders: trustedProxyHeadersEnabled(os.Getenv("TRUSTED_PROXY_HEADERS")), BlockTerminalEmail: blockTerminalEmail})
	if err != nil {
		return err
	}
	httpServer := &http.Server{Addr: envDefault("LISTEN_ADDR", ":8080"), Handler: server.Router(), ReadHeaderTimeout: 5 * time.Second}
	serveCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	traceShutdown, err := setupTracing(serveCtx)
	if err != nil {
		return err
	}
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- (email.Worker{DB: dbConn, Resolver: email.Resolver{DB: dbConn, KEK: masterKey, TerminalWriter: os.Stdout, BlockTerminal: blockTerminalEmail}, Workers: 4, BatchSize: 10, PollInterval: time.Second}).Run(serveCtx)
	}()
	var rotationDone chan error
	if len(masterKey) == cypra.MasterKeyBytes {
		rotationDone = make(chan error, 1)
		go func() {
			rotationDone <- oidc.NewRotationService(dbConn, masterKey).Run(serveCtx, time.Hour)
		}()
	}
	cleanupDone := make(chan error, 1)
	go func() {
		err := oidc.RunCleanupTicker(serveCtx, dbConn, time.Hour)
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		cleanupDone <- err
	}()
	tenantCleanupDone := make(chan error, 1)
	go func() {
		err := httpserver.RunTenantDeletionCleanupTicker(serveCtx, dbConn, time.Hour)
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		tenantCleanupDone <- err
	}()
	httpDone := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		httpDone <- err
	}()
	var runErr error
	workerDoneRead := false
	rotationDoneRead := false
	cleanupDoneRead := false
	tenantCleanupDoneRead := false
	httpDoneRead := false
	select {
	case <-serveCtx.Done():
	case runErr = <-workerDone:
		workerDoneRead = true
		stop()
	case runErr = <-rotationDone:
		rotationDoneRead = true
		stop()
	case runErr = <-cleanupDone:
		cleanupDoneRead = true
		stop()
	case runErr = <-tenantCleanupDone:
		tenantCleanupDoneRead = true
		stop()
	case runErr = <-httpDone:
		httpDoneRead = true
		stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = err
	}
	stop()
	if !workerDoneRead {
		if err := <-workerDone; err != nil && runErr == nil {
			runErr = err
		}
	}
	if rotationDone != nil && !rotationDoneRead {
		if err := <-rotationDone; err != nil && runErr == nil {
			runErr = err
		}
	}
	if !cleanupDoneRead {
		if err := <-cleanupDone; err != nil && runErr == nil {
			runErr = err
		}
	}
	if !tenantCleanupDoneRead {
		if err := <-tenantCleanupDone; err != nil && runErr == nil {
			runErr = err
		}
	}
	if !httpDoneRead {
		if err := <-httpDone; err != nil && runErr == nil {
			runErr = err
		}
	}
	if traceShutdown != nil {
		if err := traceShutdown(shutdownCtx); err != nil && runErr == nil {
			runErr = err
		}
	}
	return runErr
}

func setupTracing(ctx context.Context) (func(context.Context) error, error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return nil, nil
	}
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, fmt.Errorf("init otlp trace exporter: %w", err)
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func runAdmin(args []string) error {
	if len(args) == 0 {
		return exitError{code: 2, error: "cypra admin: command required"}
	}
	dbConn, err := openDB(envDefault("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	switch args[0] {
	case "promote":
		return adminPromote(dbConn, args[1:])
	case "invite":
		return adminInvite(dbConn, args[1:])
	case "list-instance-admins":
		return adminListInstanceAdmins(dbConn, args[1:])
	case "reset-passkey":
		return adminResetPasskey(dbConn, args[1:])
	case "reset-passwords":
		return adminResetPasswords(dbConn, args[1:])
	case "rotate-key":
		return adminRotateKey(dbConn, args[1:])
	case "rotate-master-key":
		return adminRotateMasterKey(dbConn, args[1:])
	case "revoke-tenant-tokens":
		return adminRevokeTenantTokens(dbConn, args[1:])
	case "reset-bootstrap":
		return adminResetBootstrap(dbConn)
	case "list-tenants":
		return adminListTenants(dbConn, args[1:])
	default:
		return exitError{code: 2, error: "cypra admin: unknown command " + args[0]}
	}
}

func runVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"version": version, "commit": commit})
	}
	fmt.Printf("cypra %s (%s)\n", version, commit)
	return nil
}

func adminPromote(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin promote", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	tenantArg := fs.String("tenant", "", "tenant slug or id")
	email := fs.String("email", "", "email to invite")
	role := fs.String("role", "admin", "owner, admin, or member")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	if *tenantArg == "" || *email == "" {
		return exitError{code: 2, error: "admin promote requires --tenant and --email"}
	}
	tenantID, err := resolveTenant(context.Background(), dbConn, *tenantArg)
	if err != nil {
		return err
	}
	if err := auditCLIInstanceAdmin(dbConn, "tenant_admin.recovery", "admin promote"); err != nil {
		return err
	}
	token, id, err := (invite.Service{DB: dbConn}).Issue(context.Background(), invite.IssueRequest{TenantID: &tenantID, Email: *email, Role: *role, CreatedByKind: "system"})
	if err != nil {
		return err
	}
	fmt.Printf("invite_id=%s token=%s\n", id, token)
	return nil
}

func adminInvite(dbConn *sql.DB, args []string) error {
	if len(args) != 1 {
		return exitError{code: 2, error: "admin invite requires <email>"}
	}
	var admins int
	if err := dbConn.QueryRowContext(context.Background(), `SELECT count(*) FROM instance_admins`).Scan(&admins); err != nil {
		return err
	}
	if admins == 0 {
		return exitError{code: 8, error: "admin invite requires an initialized instance; use reset-bootstrap for first admin recovery"}
	}
	if err := auditCLIInstanceAdmin(dbConn, "instance_admin.recovery", "admin invite"); err != nil {
		return err
	}
	token, id, err := (invite.Service{DB: dbConn}).Issue(context.Background(), invite.IssueRequest{Email: args[0], Role: "instance_admin", CreatedByKind: "system"})
	if err != nil {
		return err
	}
	slog.Info("instance-admin invite minted", slog.String("request_id", ""), slog.String("tenant_id", ""), slog.String("actor_id", "system"), slog.String("invite_id", id.String()), slog.String("redemption_token", token), slog.Bool("redacted-on-export", true))
	fmt.Printf("invite_id=%s token=%s\n", id, token)
	return nil
}

func adminListInstanceAdmins(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin list-instance-admins", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	if err := auditCLIInstanceAdmin(dbConn, "instance_admin.read", "admin list-instance-admins"); err != nil {
		return err
	}
	rows, err := dbConn.Query(`SELECT id, email, display_name, created_at, last_login_at, disabled_at FROM instance_admins ORDER BY email`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var admins []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var email, name string
		var created time.Time
		var lastLogin, disabled sql.NullTime
		if err := rows.Scan(&id, &email, &name, &created, &lastLogin, &disabled); err != nil {
			return err
		}
		admins = append(admins, map[string]any{"id": id, "email": email, "display_name": name, "created_at": created, "last_login_at": nullableTime(lastLogin), "disabled_at": nullableTime(disabled)})
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(admins)
	}
	for _, admin := range admins {
		fmt.Printf("%s\t%s\t%s\n", admin["id"], admin["email"], admin["display_name"])
	}
	return rows.Err()
}

func adminResetPasskey(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin reset-passkey", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	tenantArg := fs.String("tenant", "", "tenant slug or id")
	email := fs.String("email", "", "user email")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	tenantID, err := resolveTenant(context.Background(), dbConn, *tenantArg)
	if err != nil {
		return err
	}
	if err := auditCLIInstanceAdmin(dbConn, "tenant_admin.recovery", "admin reset-passkey"); err != nil {
		return err
	}
	_, err = dbConn.Exec(`DELETE FROM passkey_credentials WHERE tenant_id = $1 AND user_id = (SELECT id FROM users WHERE tenant_id = $1 AND email = $2)`, tenantID, *email)
	return err
}

func adminResetPasswords(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin reset-passwords", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	tenantArg := fs.String("tenant", "", "tenant slug or id")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	tenantID, err := resolveTenant(context.Background(), dbConn, *tenantArg)
	if err != nil {
		return err
	}
	if err := auditCLIInstanceAdmin(dbConn, "tenant_admin.recovery", "admin reset-passwords"); err != nil {
		return err
	}
	_, err = dbConn.Exec(`UPDATE password_credentials SET must_reset = true, updated_at = now() WHERE tenant_id = $1`, tenantID)
	return err
}

func adminRotateKey(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin rotate-key", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	tenantArg := fs.String("tenant", "", "tenant slug or id")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	tenantID, err := resolveTenant(context.Background(), dbConn, *tenantArg)
	if err != nil {
		return err
	}
	if err := auditCLIInstanceAdmin(dbConn, "master_key.rotation", "admin rotate-key"); err != nil {
		return err
	}
	masterKey, err := loadMasterKey()
	if err != nil {
		return err
	}
	return oidc.NewRotationService(dbConn, masterKey).ForceRotate(context.Background(), tenantID)
}

func adminRotateMasterKey(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin rotate-master-key", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	oldEnv := fs.String("old-from-env", "", "environment variable containing old master key")
	newEnv := fs.String("new-from-env", "", "environment variable containing new master key")
	resumeID := fs.String("resume", "", "rotation id to resume")
	confirmCutover := fs.Bool("confirm-cutover", false, "mark rotation cutover complete")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	service, err := cypra.NewRotationService(dbConn, encryptedColumnTargets())
	if err != nil {
		return err
	}
	if *confirmCutover {
		rotationID, err := uuid.Parse(*resumeID)
		if err != nil {
			return exitError{code: 2, error: "--confirm-cutover requires --resume=<rotation-id>"}
		}
		return service.ConfirmCutover(context.Background(), rotationID)
	}
	oldKey, oldErr := normalizeMasterKey(os.Getenv(*oldEnv))
	newKey, newErr := normalizeMasterKey(os.Getenv(*newEnv))
	if oldErr != nil || newErr != nil || len(oldKey) != cypra.MasterKeyBytes || len(newKey) != cypra.MasterKeyBytes {
		return exitError{code: 2, error: "old and new master keys must be provided via --old-from-env and --new-from-env"}
	}
	if err := auditCLIInstanceAdmin(dbConn, "master_key.rotation", "admin rotate-master-key"); err != nil {
		return err
	}
	if *resumeID != "" {
		rotationID, err := uuid.Parse(*resumeID)
		if err != nil {
			return err
		}
		return service.Resume(context.Background(), rotationID, oldKey, newKey, cypra.RotationOptions{})
	}
	rotationID, err := service.Start(context.Background(), oldKey, newKey, cypra.RotationOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("rotation_id=%s\n", rotationID)
	return nil
}

func adminRevokeTenantTokens(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin revoke-tenant-tokens", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	tenantArg := fs.String("tenant", "", "tenant slug or id")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	tenantID, err := resolveTenant(context.Background(), dbConn, *tenantArg)
	if err != nil {
		return err
	}
	if err := auditCLIInstanceAdmin(dbConn, "tenant_deletion.recovery", "admin revoke-tenant-tokens"); err != nil {
		return err
	}
	if _, err := dbConn.Exec(`UPDATE sessions SET revoked_at = now() WHERE tenant_id = $1 AND revoked_at IS NULL`, tenantID); err != nil {
		return err
	}
	_, err = dbConn.Exec(`UPDATE oidc_refresh_tokens SET revoked_at = now(), revoke_reason = 'operator_revoked' WHERE tenant_id = $1 AND revoked_at IS NULL`, tenantID)
	return err
}

func adminResetBootstrap(dbConn *sql.DB) error {
	if err := auditCLIInstanceAdmin(dbConn, "bootstrap.recovery", "admin reset-bootstrap"); err != nil {
		return err
	}
	token, err := bootstrap.NewService(dbConn, slog.Default()).RevokeSetupToken(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("setup_token=%s\n", token)
	return nil
}

func adminListTenants(dbConn *sql.DB, args []string) error {
	fs := flag.NewFlagSet("admin list-tenants", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	if err := auditCLIInstanceAdmin(dbConn, "instance_admin.read", "admin list-tenants"); err != nil {
		return err
	}
	rows, err := dbConn.Query(`SELECT t.id, t.slug, t.name, (SELECT count(*) FROM users u WHERE u.tenant_id = t.id), (SELECT count(*) FROM projects p WHERE p.tenant_id = t.id) FROM tenants t WHERE deleted_at IS NULL ORDER BY slug`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var tenants []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var slug, name string
		var users, projects int
		if err := rows.Scan(&id, &slug, &name, &users, &projects); err != nil {
			return err
		}
		tenants = append(tenants, map[string]any{"id": id, "slug": slug, "name": name, "users": users, "projects": projects})
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(tenants)
	}
	for _, tenant := range tenants {
		fmt.Printf("%s\t%s\tusers=%d\tprojects=%d\n", tenant["id"], tenant["slug"], tenant["users"], tenant["projects"])
	}
	return rows.Err()
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	out := fs.String("out", "", "output path")
	passFile := fs.String("passphrase-file", "", "passphrase file")
	passStdin := fs.Bool("passphrase-from-stdin", false, "read passphrase from stdin")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	if *out == "" {
		return exitError{code: 2, error: "export requires --out"}
	}
	passphrase, err := readPassphrase(*passFile, *passStdin)
	if err != nil {
		return err
	}
	dbConn, err := openDB(envDefault("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	content, err := exportBackup(context.Background(), dbConn, passphrase)
	if err != nil {
		return err
	}
	return os.WriteFile(*out, content, 0o600)
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	passFile := fs.String("passphrase-file", "", "passphrase file")
	passStdin := fs.Bool("passphrase-from-stdin", false, "read passphrase from stdin")
	allowResurrect := fs.Bool("allow-resurrect", false, "allow DSR resurrection")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	if fs.NArg() != 1 {
		return exitError{code: 2, error: "import requires <path>"}
	}
	passphrase, err := readPassphrase(*passFile, *passStdin)
	if err != nil {
		return err
	}
	dbConn, err := openDB(envDefault("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	var tenants int
	if err := dbConn.QueryRow(`SELECT count(*) FROM tenants`).Scan(&tenants); err != nil {
		return err
	}
	if tenants > 0 {
		return exitError{code: 8, error: "import refuses to overwrite a non-empty Cypra DB"}
	}
	content, err := os.ReadFile(fs.Arg(0)) // #nosec G304 -- operator-provided import path.
	if err != nil {
		return err
	}
	return importBackup(context.Background(), dbConn, content, passphrase, *allowResurrect)
}

func readPassphrase(file string, stdin bool) (string, error) {
	if file == "" && !stdin {
		return "", exitError{code: 7, error: "passphrase required; use --passphrase-file or --passphrase-from-stdin"}
	}
	if file != "" {
		content, err := os.ReadFile(file) // #nosec G304 -- operator-provided passphrase path.
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(content)), nil
	}
	var passphrase string
	if _, err := fmt.Fscan(os.Stdin, &passphrase); err != nil {
		return "", err
	}
	return passphrase, nil
}

func resolveTenant(ctx context.Context, dbConn *sql.DB, tenant string) (uuid.UUID, error) {
	if tenant == "" {
		return uuid.Nil, exitError{code: 2, error: "--tenant is required"}
	}
	if id, err := uuid.Parse(tenant); err == nil {
		return id, nil
	}
	var id uuid.UUID
	err := dbConn.QueryRowContext(ctx, `SELECT id FROM tenants WHERE slug = $1 AND deleted_at IS NULL`, tenant).Scan(&id)
	return id, err
}

func auditCLIInstanceAdmin(dbConn *sql.DB, action, reason string) error {
	_, err := dbConn.ExecContext(
		context.Background(),
		`INSERT INTO audit_entries (tenant_id, actor_kind, actor_id, action, resource_kind, metadata)
		 VALUES (NULL, 'instance_admin', NULL, $1::text, 'cli_admin', jsonb_build_object('cross_tenant', true, 'reason', $2::text))`,
		action,
		reason,
	)
	return err
}

func nullableTime(value sql.NullTime) any {
	if value.Valid {
		return value.Time
	}
	return nil
}

func runMigrate() error {
	dsn := os.Getenv("MIGRATE_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
		fmt.Fprintln(os.Stderr, "warning: MIGRATE_DATABASE_URL unset; falling back to DATABASE_URL")
	}
	dbConn, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	return migrate.Apply(context.Background(), dbConn, "")
}

func openDB(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return dbConn, nil
}

func validatePublicBaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("PUBLIC_BASE_URL must be a valid URL")
	}
	if parsed.Scheme != "https" && os.Getenv("CYPRA_DEV_INSECURE_HTTP") != "true" {
		return fmt.Errorf("PUBLIC_BASE_URL must be HTTPS outside dev")
	}
	return nil
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func masterKeyConfigured() bool {
	return strings.TrimSpace(os.Getenv("MASTER_KEY")) != "" || strings.TrimSpace(os.Getenv("MASTER_KEY_FILE")) != ""
}

func trustedProxyHeadersEnabled(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value != "" && value != "none"
}

func terminalEmailBlocked(publicBaseURL string) bool {
	parsed, err := url.Parse(publicBaseURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	return host != "localhost" && host != "127.0.0.1" && host != "::1" && !strings.HasSuffix(host, ".localhost")
}

func loadMasterKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("MASTER_KEY"))
	if file := strings.TrimSpace(os.Getenv("MASTER_KEY_FILE")); raw == "" && file != "" {
		content, err := os.ReadFile(file) // #nosec G304,G703 -- operator-provided key file path.
		if err != nil {
			return nil, fmt.Errorf("read MASTER_KEY_FILE: %w", err)
		}
		raw = strings.TrimSpace(string(content))
	}
	return normalizeMasterKey(raw)
}

func normalizeMasterKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, nil
	}
	if len([]byte(raw)) == cypra.MasterKeyBytes && weakRawMasterKey(raw) {
		return nil, cypra.ErrInvalidMasterKey
	}
	if key, err := cypra.LoadMasterKey(raw, ""); err == nil {
		return key, nil
	}
	return nil, cypra.ErrInvalidMasterKey
}

func weakRawMasterKey(raw string) bool {
	first := raw[0]
	for i := 1; i < len(raw); i++ {
		if raw[i] != first {
			return false
		}
	}
	return true
}

func encryptedColumnTargets() []cypra.EncryptedColumn {
	return []cypra.EncryptedColumn{
		{Table: "oidc_signing_keys", IDColumn: "id", DEKColumn: "private_key_encrypted", Format: cypra.EncryptedColumnFormatJSONEnvelope},
		{Table: "oidc_clients", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "upstream_providers", IDColumn: "id", DEKColumn: "client_id_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "upstream_providers", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "social_connections", IDColumn: "id", DEKColumn: "client_id_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "social_connections", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "oidc_connections", IDColumn: "id", DEKColumn: "client_id_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "oidc_connections", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "email_provider_configs", IDColumn: "id", DEKColumn: "config_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "totp_credentials", IDColumn: "id", DEKColumn: "secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
	}
}

type exitError struct {
	code  int
	error string
}

func (e exitError) Error() string {
	return e.error
}
