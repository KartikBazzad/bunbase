package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:8080"

func main() {
	baseURL := flag.String("url", defaultBaseURL, "BunKMS base URL")
	token := flag.String("token", os.Getenv("BUNKMS_TOKEN"), "JWT Bearer token (or BUNKMS_TOKEN)")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	cmd, rest := args[0], args[1:]
	client := &client{baseURL: strings.TrimSuffix(*baseURL, "/"), token: *token}
	if err := run(client, cmd, rest); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `bunkms-cli - BunKMS command-line client

Usage: bunkms-cli [flags] <command> [args]

Commands:
  key create <name> [type]     Create a key (type: aes-256, rsa-2048, ecdsa-p256; default aes-256)
  key get <name>               Get key metadata
  key rotate <name>            Rotate key
  key revoke <name>            Revoke key
  encrypt <key> <plaintext>    Encrypt with key (plaintext or - for stdin)
  decrypt <key> <ciphertext>  Decrypt (ciphertext base64 or - for stdin)
  secret put <name> <value>   Store a secret
  secret get <name>            Retrieve a secret

Flags:
  -url string   BunKMS base URL (default %s)
  -token string JWT Bearer token (or set BUNKMS_TOKEN)

`, defaultBaseURL)
}

type client struct {
	baseURL string
	token   string
}

func (c *client) do(method, path string, body []byte) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+path, r)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")
	hc := &http.Client{Timeout: 15 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return out, resp.StatusCode, nil
}

func run(cl *client, cmd string, args []string) error {
	switch cmd {
	case "key":
		if len(args) < 2 {
			return fmt.Errorf("key subcommand and name required")
		}
		return runKey(cl, args[0], args[1], args[2:])
	case "encrypt":
		if len(args) < 2 {
			return fmt.Errorf("encrypt <key> <plaintext> required")
		}
		return runEncrypt(cl, args[0], args[1])
	case "decrypt":
		if len(args) < 2 {
			return fmt.Errorf("decrypt <key> <ciphertext_b64> required")
		}
		return runDecrypt(cl, args[0], args[1])
	case "secret":
		if len(args) < 2 {
			return fmt.Errorf("secret put|get <name> [value] required")
		}
		return runSecret(cl, args[0], args[1], args[2:])
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func runKey(cl *client, sub, name string, rest []string) error {
	switch sub {
	case "create":
		keyType := "aes-256"
		if len(rest) > 0 {
			keyType = rest[0]
		}
		body, _ := json.Marshal(map[string]string{"name": name, "type": keyType})
		out, code, err := cl.do("POST", "/v1/keys", body)
		if err != nil {
			return err
		}
		if code != http.StatusCreated {
			return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
		}
		fmt.Println(string(out))
		return nil
	case "get":
		out, code, err := cl.do("GET", "/v1/keys/"+name, nil)
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
		}
		fmt.Println(string(out))
		return nil
	case "rotate":
		out, code, err := cl.do("POST", "/v1/keys/"+name+"/rotate", nil)
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
		}
		fmt.Println(string(out))
		return nil
	case "revoke":
		out, code, err := cl.do("POST", "/v1/keys/"+name+"/revoke", nil)
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
		}
		fmt.Println(string(out))
		return nil
	default:
		return fmt.Errorf("unknown key subcommand %q", sub)
	}
}

func runEncrypt(cl *client, keyName, plaintext string) error {
	if plaintext == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		plaintext = string(b)
	}
	body, _ := json.Marshal(map[string]string{"plaintext": plaintext})
	out, code, err := cl.do("POST", "/v1/encrypt/"+keyName, body)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
	}
	var res struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return err
	}
	fmt.Println(res.Ciphertext)
	return nil
}

func runDecrypt(cl *client, keyName, ciphertext string) error {
	if ciphertext == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		ciphertext = string(bytes.TrimSpace(b))
	}
	body, _ := json.Marshal(map[string]string{"ciphertext": ciphertext})
	out, code, err := cl.do("POST", "/v1/decrypt/"+keyName, body)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
	}
	var res struct {
		Plaintext    string `json:"plaintext"`
		PlaintextB64 string `json:"plaintext_b64"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return err
	}
	if res.Plaintext != "" {
		fmt.Print(res.Plaintext)
	} else {
		b, _ := base64.StdEncoding.DecodeString(res.PlaintextB64)
		os.Stdout.Write(b)
	}
	return nil
}

func runSecret(cl *client, sub, name string, rest []string) error {
	switch sub {
	case "put":
		if len(rest) < 1 {
			return fmt.Errorf("secret put <name> <value> required")
		}
		value := rest[0]
		var body []byte
		if value == "-" {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			body, _ = json.Marshal(map[string]string{"name": name, "value_b64": base64.StdEncoding.EncodeToString(b)})
		} else {
			body, _ = json.Marshal(map[string]string{"name": name, "value": value})
		}
		out, code, err := cl.do("POST", "/v1/secrets", body)
		if err != nil {
			return err
		}
		if code != http.StatusCreated {
			return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
		}
		fmt.Println(string(out))
		return nil
	case "get":
		out, code, err := cl.do("GET", "/v1/secrets/"+name, nil)
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("%s: %s", http.StatusText(code), string(out))
		}
		var res struct {
			Value    string `json:"value"`
			ValueB64 string `json:"value_b64"`
		}
		if err := json.Unmarshal(out, &res); err != nil {
			return err
		}
		if res.Value != "" {
			fmt.Print(res.Value)
		} else {
			b, _ := base64.StdEncoding.DecodeString(res.ValueB64)
			os.Stdout.Write(b)
		}
		return nil
	default:
		return fmt.Errorf("unknown secret subcommand %q", sub)
	}
}
