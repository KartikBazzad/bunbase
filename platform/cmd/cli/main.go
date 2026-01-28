package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type Config struct {
	BaseURL       string `json:"base_url"`
	ProjectID     string `json:"project_id"`
	SessionCookie string `json:"session_cookie"` // e.g. "session_token=abc..."
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".bunbase")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "cli_config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// --- Cobra root and top-level commands ---

var rootCmd = &cobra.Command{
	Use:   "bunbase",
	Short: "BunBase CLI",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---- Login ----

func cmdLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "Email address")
	password := fs.String("password", "", "Password")
	baseURL := fs.String("base-url", "http://localhost:3001/api", "Platform API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *email == "" || *password == "" {
		return fmt.Errorf("email and password are required")
	}

	body, err := json.Marshal(map[string]string{
		"email":    *email,
		"password": *password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(*baseURL, "/")+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", strings.TrimSpace(string(msg)))
	}

	// Extract session_token cookie
	var sessionCookie string
	for _, c := range resp.Cookies() {
		if c.Name == "session_token" {
			sessionCookie = c.Name + "=" + c.Value
			break
		}
	}
	if sessionCookie == "" {
		return fmt.Errorf("no session_token cookie received")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.BaseURL = strings.TrimRight(*baseURL, "/")
	cfg.SessionCookie = sessionCookie
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Println("✅ Logged in successfully")
	return nil
}

// ---- Projects ----

func cmdProjects(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: bunbase projects [list|create|use]")
		return nil
	}

	sub := args[0]
	switch sub {
	case "list":
		return projectsList(args[1:])
	case "create":
		return projectsCreate(args[1:])
	case "use":
		return projectsUse(args[1:])
	default:
		fmt.Println("Usage: bunbase projects [list|create|use]")
		return nil
	}
}

func requireAuthConfig() (*Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.BaseURL == "" || cfg.SessionCookie == "" {
		return nil, fmt.Errorf("not logged in; run `bunbase login` first")
	}
	return cfg, nil
}

func doAuthedRequest(cfg *Config, method, path string, body io.Reader) (*http.Response, error) {
	url := strings.TrimRight(cfg.BaseURL, "/") + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cfg.SessionCookie)
	client := &http.Client{}
	return client.Do(req)
}

func projectsList(args []string) error {
	_ = args
	cfg, err := requireAuthConfig()
	if err != nil {
		return err
	}

	resp, err := doAuthedRequest(cfg, http.MethodGet, "/projects", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("list projects failed: %s", strings.TrimSpace(string(msg)))
	}

	var projects []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects found. Create one with `bunbase projects create <name>`.")
		return nil
	}

	fmt.Println("Projects:")
	for _, p := range projects {
		active := ""
		if cfg.ProjectID == p.ID {
			active = " (active)"
		}
		fmt.Printf("  %s%s\n    ID: %s\n    Slug: %s\n", p.Name, active, p.ID, p.Slug)
	}
	return nil
}

// ---- Cobra command wiring ----

func init() {
	// login
	loginCmd := &cobra.Command{
		Use:                "login",
		Short:              "Login to BunBase Platform",
		DisableFlagParsing: true, // delegate flag parsing to cmdLogin (uses flag package)
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdLogin(args)
		},
	}

	// projects
	projectsCmd := &cobra.Command{
		Use:                "projects",
		Short:              "Manage projects",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdProjects(args)
		},
	}

	// functions
	functionsCmd := &cobra.Command{
		Use:                "functions",
		Short:              "Manage functions",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdFunctions(args)
		},
	}

	// dev
	devCmd := &cobra.Command{
		Use:                "dev",
		Short:              "Run local dev server for functions",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdDev(args)
		},
	}

	// whoami
	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently logged in user",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireAuthConfig()
			if err != nil {
				return err
			}

			url := strings.TrimRight(cfg.BaseURL, "/") + "/auth/me"
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Cookie", cfg.SessionCookie)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("request failed: %s", strings.TrimSpace(string(body)))
			}

			var user struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Name  string `json:"name"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
				return err
			}

			fmt.Println("Current user:")
			fmt.Printf("  ID:    %s\n", user.ID)
			fmt.Printf("  Name:  %s\n", user.Name)
			fmt.Printf("  Email: %s\n", user.Email)
			return nil
		},
	}

	rootCmd.AddCommand(loginCmd, projectsCmd, functionsCmd, devCmd, whoamiCmd)
}

func projectsCreate(args []string) error {
	fs := flag.NewFlagSet("projects create", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: bunbase projects create <name>")
	}
	name := strings.Join(fs.Args(), " ")

	cfg, err := requireAuthConfig()
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return err
	}

	resp, err := doAuthedRequest(cfg, http.MethodPost, "/projects", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create project failed: %s", strings.TrimSpace(string(msg)))
	}

	var project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return err
	}

	cfg.ProjectID = project.ID
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("✅ Created project %s (ID: %s, Slug: %s) and set as active\n", project.Name, project.ID, project.Slug)
	return nil
}

func projectsUse(args []string) error {
	fs := flag.NewFlagSet("projects use", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: bunbase projects use <project-id>")
	}
	id := fs.Arg(0)

	cfg, err := requireAuthConfig()
	if err != nil {
		return err
	}

	resp, err := doAuthedRequest(cfg, http.MethodGet, "/projects/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("get project failed: %s", strings.TrimSpace(string(msg)))
	}

	cfg.ProjectID = id
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("✅ Active project set to %s\n", id)
	return nil
}

// ---- Functions ----

func cmdFunctions(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  bunbase functions deploy --file <path> [--name <name>] [--runtime <runtime>] [--handler <handler>] [--version <version>]")
		fmt.Println("  bunbase functions init <directory> [--template=ts|js]")
		return nil
	}

	sub := args[0]
	switch sub {
	case "deploy":
		return functionsDeploy(args[1:])
	case "init":
		return functionsInit(args[1:])
	default:
		fmt.Println("Usage:")
		fmt.Println("  bunbase functions deploy --file <path> [--name <name>] [--runtime <runtime>] [--handler <handler>] [--version <version>]")
		fmt.Println("  bunbase functions init <directory> [--template=ts|js]")
		return nil
	}
}

func functionsInit(args []string) error {
	fs := flag.NewFlagSet("functions init", flag.ExitOnError)
	template := fs.String("template", "ts", "Template language (ts or js)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: bunbase functions init <directory> [--template=ts|js]")
	}

	dir := fs.Arg(0)
	tmpl := strings.ToLower(*template)
	if tmpl != "ts" && tmpl != "js" {
		return fmt.Errorf("invalid template %q; expected \"ts\" or \"js\"", tmpl)
	}

	rootDir := dir
	if !filepath.IsAbs(rootDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		rootDir = filepath.Join(cwd, dir)
	}

	if err := ensureNewDir(rootDir); err != nil {
		return err
	}

	name := filepath.Base(dir)
	var files templateFileMap
	if tmpl == "ts" {
		files = getFunctionTemplateTs(name)
	} else {
		files = getFunctionTemplateJs(name)
	}

	if err := writeTemplateFiles(rootDir, files); err != nil {
		return err
	}

	fmt.Printf("✅ Created function template in ./%s\n\n", dir)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", dir)
	fmt.Println("  bun install        # if you add dependencies")
	if tmpl == "ts" {
		fmt.Println("  bun run src/index.ts   # run the function locally")
		fmt.Println("  bun build src/index.ts --outdir=dist --target=bun   # build to dist/")
	} else {
		fmt.Println("  bun run src/index.js   # run the function locally")
	}
	fmt.Println()
	fmt.Println("Then deploy with (example):")
	fmt.Println("  bunbase functions deploy --file dist/index.js --name", name, "--runtime bun")

	return nil
}

// ---- Dev (local helper) ----

// cmdDev is a convenience wrapper around functionsDeploy that also prints a
// local HTTP URL for testing via the functions HTTP gateway.
//
// It does NOT start the functions service for you – you still need the
// BunBase Functions binary running with the HTTP gateway enabled
// (by default on http://localhost:8080).
func cmdDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	entry := fs.String("entry", "", "Path to function entry/bundle (defaults to dist/index.js or src/index.ts|js)")
	name := fs.String("name", "", "Function name (defaults to directory name)")
	runtime := fs.String("runtime", "bun", "Runtime (bun or quickjs-ng)")
	handler := fs.String("handler", "default", "Handler name")
	port := fs.Int("port", 8787, "Port for local dev HTTP server")
	bin := fs.String("runner", "functions-dev", "Dev runner binary name or path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Derive entry path if not provided (mirror functions-dev defaults).
	if *entry == "" {
		candidates := []string{
			"dist/index.js",
			"src/index.ts",
			"src/index.js",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				*entry = c
				break
			}
		}
		if *entry == "" {
			return fmt.Errorf("could not find an entry file (tried dist/index.js, src/index.ts, src/index.js); pass --entry explicitly")
		}
	}

	// Derive name from flag or directory.
	fnName := *name
	if fnName == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		fnName = filepath.Base(wd)
	}

	// Build args for functions-dev.
	argsDev := []string{
		"--entry", *entry,
		"--name", fnName,
		"--runtime", *runtime,
		"--handler", *handler,
		"--port", fmt.Sprintf("%d", *port),
	}

	cmd := exec.Command(*bin, argsDev...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Starting bunbase dev using %s on http://127.0.0.1:%d/functions/%s\n", *bin, *port, fnName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dev runner exited with error: %w", err)
	}

	return nil
}

func functionsDeploy(args []string) error {
	fs := flag.NewFlagSet("functions deploy", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to function source file (.ts or .js)")
	name := fs.String("name", "", "Function name (defaults to filename)")
	runtime := fs.String("runtime", "bun", "Runtime (bun or quickjs-ng)")
	handler := fs.String("handler", "default", "Handler name")
	version := fs.String("version", "v1", "Version tag")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *filePath == "" {
		return fmt.Errorf("required flag: --file <path>")
	}

	cfg, err := requireAuthConfig()
	if err != nil {
		return err
	}
	if cfg.ProjectID == "" {
		return fmt.Errorf("no active project; run `bunbase projects use <project-id>` first")
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	fnName := *name
	if fnName == "" {
		base := filepath.Base(*filePath)
		fnName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	encoded := encodeBase64(data)

	body, err := json.Marshal(map[string]string{
		"name":    fnName,
		"runtime": *runtime,
		"handler": *handler,
		"version": *version,
		"bundle":  encoded,
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/functions", cfg.ProjectID)
	resp, err := doAuthedRequest(cfg, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deploy failed: %s", strings.TrimSpace(string(msg)))
	}

	var fn struct {
		ID               string `json:"id"`
		FunctionServiceID string `json:"function_service_id"`
		Name             string `json:"name"`
		Runtime          string `json:"runtime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fn); err != nil {
		return err
	}

	fmt.Println("✅ Function deployed")
	fmt.Printf("   Name: %s\n", fn.Name)
	fmt.Printf("   ID: %s\n", fn.ID)
	fmt.Printf("   Service ID: %s\n", fn.FunctionServiceID)
	fmt.Printf("   Runtime: %s\n", fn.Runtime)
	return nil
}

// Minimal base64 encoding without extra deps
const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func encodeBase64(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var buf strings.Builder
	rem := len(data) % 3
	mainLen := len(data) - rem

	for i := 0; i < mainLen; i += 3 {
		b := (uint(data[i]) << 16) | (uint(data[i+1]) << 8) | uint(data[i+2])
		buf.WriteByte(base64Table[(b>>18)&0x3F])
		buf.WriteByte(base64Table[(b>>12)&0x3F])
		buf.WriteByte(base64Table[(b>>6)&0x3F])
		buf.WriteByte(base64Table[b&0x3F])
	}

	if rem == 1 {
		b := uint(data[mainLen]) << 16
		buf.WriteByte(base64Table[(b>>18)&0x3F])
		buf.WriteByte(base64Table[(b>>12)&0x3F])
		buf.WriteByte('=')
		buf.WriteByte('=')
	} else if rem == 2 {
		b := (uint(data[mainLen]) << 16) | (uint(data[mainLen+1]) << 8)
		buf.WriteByte(base64Table[(b>>18)&0x3F])
		buf.WriteByte(base64Table[(b>>12)&0x3F])
		buf.WriteByte(base64Table[(b>>6)&0x3F])
		buf.WriteByte('=')
	}

	return buf.String()
}

// ---- Local function templates (TS/JS) ----

type templateFileMap map[string]string

func ensureNewDir(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("directory %q already exists", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if _, err := os.Stat(dir); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

func writeTemplateFiles(root string, files templateFileMap) error {
	for rel, contents := range files {
		target := filepath.Join(root, rel)
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("refusing to overwrite existing file %q", target)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := ensureParentDir(target); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(contents), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func getFunctionTemplateTs(name string) templateFileMap {
	packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "bun build src/index.ts --outdir=dist",
    "dev": "bun run src/index.ts"
  }
}
`, name)

	indexTS := `export async function handler(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const name = url.searchParams.get("name") ?? "World";

  return new Response(` + "`" + `Hello, ${name} from BunBase function!
` + "`" + `, {
    status: 200,
    headers: { "Content-Type": "text/plain" },
  });
}
`

	readme := fmt.Sprintf(`# %s

This directory was generated by "bunbase functions init".

## Files

- src/index.ts - Entry point exporting handler(request: Request): Promise<Response>.
- package.json - Minimal Bun-friendly package with build and dev scripts.

## Local development

  bun install       # only if you add dependencies
  bun run dev       # run the function directly with Bun

## Build for deployment

  bun run build

Then deploy the built file with the BunBase CLI, for example:

  bunbase functions deploy \\
    --file dist/index.js \\
    --name %s \\
    --runtime bun
`, name, name)

	return templateFileMap{
		"package.json": packageJSON,
		"src/index.ts": indexTS,
		"README.md":    readme,
		".gitignore": `node_modules
dist
.DS_Store
*.log
`,
	}
}

func getFunctionTemplateJs(name string) templateFileMap {
	packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "bun run src/index.js"
  }
}
`, name)

	indexJS := `export async function handler(request) {
  const url = new URL(request.url);
  const name = url.searchParams.get("name") ?? "World";

  return new Response(` + "`" + `Hello, ${name} from BunBase function!
` + "`" + `, {
    status: 200,
    headers: { "Content-Type": "text/plain" },
  });
}
`

	readme := fmt.Sprintf(`# %s

This directory was generated by "bunbase functions init".

## Files

- src/index.js - Entry point exporting handler(request).
- package.json - Minimal Bun-friendly package with a dev script.

## Local development

  bun install       # only if you add dependencies
  bun run dev       # run the function directly with Bun

For more complex setups you can add a build step using "bun build" similar
to the TypeScript template.
`, name)

	return templateFileMap{
		"package.json": packageJSON,
		"src/index.js": indexJS,
		"README.md":    readme,
		".gitignore": `node_modules
dist
.DS_Store
*.log
`,
	}
}

