/*
 * QuickJS-NG Worker
 * 
 * This program embeds QuickJS-NG and libuv to execute JavaScript functions
 * in a sandboxed environment with capability-based security.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/resource.h>
#include <errno.h>
#include <stdint.h>

// QuickJS-NG headers
#include "quickjs.h"
#include "quickjs-libc.h"
#include "cutils.h"

// libuv headers
#include "uv.h"

#define MAX_LINE_LENGTH 1024 * 1024  // 1MB max line length for NDJSON
#define MAX_BUNDLE_SIZE 10 * 1024 * 1024  // 10MB max bundle size

// Global state
static JSContext *ctx = NULL;
static JSRuntime *rt = NULL;
static JSValue handler_func = JS_UNDEFINED;
static char *bundle_path = NULL;
static char worker_id[64] = {0};
/* Current invocation ID; set at start of execute_handler, cleared at end. Used by console override. */
static char current_invoke_id[64] = {0};

// Capabilities structure
typedef struct {
    int allow_filesystem;
    int allow_network;
    int allow_child_process;
    int allow_eval;
    long max_memory;
    int max_fds;
} capabilities_t;

static capabilities_t caps = {0};

// Forward declarations
static void send_ready(void);
static void send_error(const char *id, const char *message, const char *code);
static void send_response(const char *id, int status, const char *headers_json, const char *body_base64);
static void send_log(const char *id, const char *level, const char *message);
static int load_bundle(const char *path);
static int execute_handler(const char *invoke_id, const char *method, const char *path, 
                          const char *headers_json, const char *query_json, const char *body_base64);
static void setup_capabilities(void);
static void enforce_resource_limits(void);
static void add_web_apis(JSContext *ctx);
static void add_console_override(JSContext *ctx);

// Send NDJSON message to stdout
static void send_message(const char *type, const char *id, const char *payload) {
    printf("{\"id\":\"%s\",\"type\":\"%s\",\"payload\":%s}\n", id, type, payload);
    fflush(stdout);
}

static void send_ready(void) {
    send_message("ready", worker_id, "{}");
}

static void send_error(const char *id, const char *message, const char *code) {
    char payload[1024];
    snprintf(payload, sizeof(payload), 
             "{\"message\":\"%s\",\"code\":\"%s\"}", 
             message ? message : "Unknown error",
             code ? code : "UNKNOWN_ERROR");
    send_message("error", id, payload);
}

static void send_response(const char *id, int status, const char *headers_json, const char *body_base64) {
    // Escape JSON string for body (simple escaping)
    char escaped_body[16384] = "";
    if (body_base64 && strlen(body_base64) > 0) {
        // Simple JSON string escaping
        size_t j = 0;
        for (size_t i = 0; i < strlen(body_base64) && j < sizeof(escaped_body) - 1; i++) {
            char c = body_base64[i];
            if (c == '"') {
                escaped_body[j++] = '\\';
                escaped_body[j++] = '"';
            } else if (c == '\\') {
                escaped_body[j++] = '\\';
                escaped_body[j++] = '\\';
            } else if (c == '\n') {
                escaped_body[j++] = '\\';
                escaped_body[j++] = 'n';
            } else if (c == '\r') {
                escaped_body[j++] = '\\';
                escaped_body[j++] = 'r';
            } else {
                escaped_body[j++] = c;
            }
        }
        escaped_body[j] = '\0';
    }
    
    char payload[32768];  // Increased size for larger responses
    snprintf(payload, sizeof(payload),
             "{\"status\":%d,\"headers\":%s,\"body\":\"%s\"}",
             status,
             headers_json ? headers_json : "{}",
             escaped_body);
    send_message("response", id, payload);
}

/* Escape string for JSON and truncate to fit in out_buf (includes null). Returns out_buf. */
static char *escape_json_str(const char *in, char *out_buf, size_t out_size) {
    if (!out_size) return out_buf;
    size_t j = 0;
    for (const char *p = in; p && *p && j < out_size - 1; p++) {
        if (*p == '"' || *p == '\\') {
            if (j + 2 >= out_size) break;
            out_buf[j++] = '\\';
            out_buf[j++] = *p;
        } else if (*p == '\n') {
            if (j + 2 >= out_size) break;
            out_buf[j++] = '\\';
            out_buf[j++] = 'n';
        } else if (*p == '\r') {
            if (j + 2 >= out_size) break;
            out_buf[j++] = '\\';
            out_buf[j++] = 'r';
        } else {
            out_buf[j++] = *p;
        }
    }
    out_buf[j] = '\0';
    return out_buf;
}

static void send_log(const char *id, const char *level, const char *message) {
    char msg_escaped[768];
    escape_json_str(message ? message : "", msg_escaped, sizeof(msg_escaped));
    char payload[1024];
    snprintf(payload, sizeof(payload),
             "{\"level\":\"%s\",\"message\":\"%s\"}",
             level ? level : "info",
             msg_escaped);
    send_message("log", id, payload);
}

// Setup capabilities from environment
static void setup_capabilities(void) {
    const char *caps_json = getenv("CAPABILITIES");
    if (caps_json) {
        // Parse JSON capabilities (simplified - in production use proper JSON parser)
        // For now, use environment variables directly
    }
    
    // Parse individual capability flags
    caps.allow_filesystem = getenv("ALLOW_FILESYSTEM") != NULL;
    caps.allow_network = getenv("ALLOW_NETWORK") != NULL;
    caps.allow_child_process = getenv("ALLOW_CHILD_PROCESS") != NULL;
    caps.allow_eval = getenv("ALLOW_EVAL") != NULL;
    
    const char *max_mem = getenv("MAX_MEMORY");
    if (max_mem) {
        caps.max_memory = atol(max_mem);
    }
    
    const char *max_fds = getenv("MAX_FDS");
    if (max_fds) {
        caps.max_fds = atoi(max_fds);
    }
}

// Missing cutils symbol override


// Add atob/btoa polyfills
static void add_base64_polyfills(JSContext *ctx) {
    const char *base64_polyfill = 
        "(function() {"
        "  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';"
        "  globalThis.btoa = function(input) {"
        "    let str = String(input);"
        "    let output = '';"
        "    for (let block = 0, charCode, i = 0, map = chars; "
        "    str.charAt(i | 0) || (map = '=', i % 1); "
        "    output += map.charAt(63 & block >> 8 - i % 1 * 8)) {"
        "      charCode = str.charCodeAt(i += 3/4);"
        "      if (charCode > 0xFF) {"
        "        throw new Error(\"'btoa' failed: The string to be encoded contains characters outside of the Latin1 range.\");"
        "      }"
        "      block = block << 8 | charCode;"
        "    }"
        "    return output;"
        "  };"
        "  globalThis.atob = function(input) {"
        "    let str = String(input).replace(/=+$/, '');"
        "    let output = '';"
        "    if (str.length % 4 == 1) {"
        "      throw new Error(\"'atob' failed: The string to be decoded is not correctly encoded.\");"
        "    }"
        "    for (let bc = 0, bs = 0, buffer, i = 0; "
        "      buffer = str.charAt(i++); "
        "      ~buffer && (bs = bc % 4 ? bs * 64 + buffer : buffer, "
        "        bc++ % 4) ? output += String.fromCharCode(255 & bs >> (-2 * bc & 6)) : 0"
        "    ) {"
        "      buffer = chars.indexOf(buffer);"
        "    }"
        "    return output;"
        "  };"
        "})();";

    JSValue result = JS_Eval(ctx, base64_polyfill, strlen(base64_polyfill), "<base64-polyfill>", JS_EVAL_TYPE_GLOBAL);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[WARN] Failed to add Base64 polyfills: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
    } else {
        JS_FreeValue(ctx, result);
    }
}

// Add Web API polyfills (URL, Response, Request)
static void add_web_apis(JSContext *ctx) {
    // Add URL polyfill
    const char *url_polyfill =
        "(function() {"
        "  class URLSearchParams {"
        "    constructor(init) {"
        "      this.params = {};"
        "      if (typeof init === 'string') {"
        "        if (init) {"
        "          init.split('&').forEach(pair => {"
        "            const eq = pair.indexOf('=');"
        "            if (eq >= 0) {"
        "              const key = decodeURIComponent(pair.substring(0, eq));"
        "              const value = decodeURIComponent(pair.substring(eq + 1));"
        "              this.params[key] = value;"
        "            } else if (pair) {"
        "              this.params[decodeURIComponent(pair)] = '';"
        "            }"
        "          });"
        "        }"
        "      } else if (init) {"
        "        Object.entries(init).forEach(([k, v]) => this.params[k] = v);"
        "      }"
        "    }"
        "    get(name) { return this.params[name] || null; }"
        "    set(name, value) { this.params[name] = value; }"
        "    has(name) { return name in this.params; }"
        "    delete(name) { delete this.params[name]; }"
        "    forEach(callback) { Object.entries(this.params).forEach(([k, v]) => callback(v, k)); }"
        "    entries() { return Object.entries(this.params); }"
        "    keys() { return Object.keys(this.params); }"
        "    values() { return Object.values(this.params); }"
        "  }"
        "  class URL {"
        "    constructor(url, base) {"
        "      let fullUrl = url;"
        "      if (base) {"
        "        if (typeof base === 'string') {"
        "          const baseUrl = new URL(base);"
        "          if (url.startsWith('/')) {"
        "            fullUrl = baseUrl.origin + url;"
        "          } else {"
        "            fullUrl = baseUrl.href.replace(/\\/[^/]*$/, '/') + url;"
        "          }"
        "        } else {"
        "          fullUrl = base.href + url;"
        "        }"
        "      }"
        "      this.href = fullUrl;"
        "      const match = fullUrl.match(/^(https?:\\/\\/[^\\/]+)?([^?#]*)(\\?[^#]*)?(#.*)?$/);"
        "      this.origin = match && match[1] ? match[1] : '';"
        "      this.pathname = match && match[2] ? match[2] : '/';"
        "      const search = match && match[3] ? match[3] : '';"
        "      this.search = search;"
        "      this.hash = match && match[4] ? match[4] : '';"
        "      this.searchParams = new URLSearchParams(search.substring(1));"
        "    }"
        "    toString() { "
        "      const pairs = []; "
        "      Object.entries(this.searchParams.params).forEach(([k, v]) => { "
        "        pairs.push(encodeURIComponent(k) + (v ? '=' + encodeURIComponent(v) : '')); "
        "      }); "
        "      this.search = pairs.length > 0 ? '?' + pairs.join('&') : ''; "
        "      this.href = this.origin + this.pathname + this.search + this.hash; "
        "      return this.href; "
        "    }"
        "  }"
        "  globalThis.URL = URL;"
        "  globalThis.URLSearchParams = URLSearchParams;"
        "})();";
    
    JSValue result = JS_Eval(ctx, url_polyfill, strlen(url_polyfill), "<url-polyfill>", JS_EVAL_TYPE_GLOBAL);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[WARN] Failed to add URL polyfill: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
    } else {
        JS_FreeValue(ctx, result);
    }
    
    // Add Response polyfill
    const char *response_polyfill =
        "(function() {"
        "  class Headers {"
        "    constructor(init) {"
        "      this._headers = {};"
        "      if (init) {"
        "        if (typeof init === 'object' && !Array.isArray(init)) {"
        "          Object.entries(init).forEach(([k, v]) => this._headers[k.toLowerCase()] = String(v));"
        "        }"
        "      }"
        "    }"
        "    get(name) { return this._headers[name.toLowerCase()] || null; }"
        "    set(name, value) { this._headers[name.toLowerCase()] = String(value); }"
        "    has(name) { return name.toLowerCase() in this._headers; }"
        "    delete(name) { delete this._headers[name.toLowerCase()]; }"
        "    forEach(callback) { Object.entries(this._headers).forEach(([k, v]) => callback(v, k)); }"
        "    get headers() { return this._headers; }"
        "  }"
        "  class Response {"
        "    constructor(body, init) {"
        "      this.body = body || null;"
        "      this.status = (init && init.status) || 200;"
        "      this.statusText = (init && init.statusText) || 'OK';"
        "      this.headers = new Headers(init && init.headers);"
        "      this.ok = this.status >= 200 && this.status < 300;"
        "    }"
        "    static json(data) {"
        "      const bodyStr = JSON.stringify(data);"
        "      return new Response(bodyStr, {"
        "        headers: { 'Content-Type': 'application/json' }"
        "      });"
        "    }"
        "    static text(text) {"
        "      return new Response(String(text), {"
        "        headers: { 'Content-Type': 'text/plain' }"
        "      });"
        "    }"
        "  }"
        "  globalThis.Response = Response;"
        "  globalThis.Headers = Headers;"
        "})();";
    
    result = JS_Eval(ctx, response_polyfill, strlen(response_polyfill), "<response-polyfill>", JS_EVAL_TYPE_GLOBAL);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[WARN] Failed to add Response polyfill: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
    } else {
        JS_FreeValue(ctx, result);
    }
    
    // Add Request polyfill (simplified)
    const char *request_polyfill =
        "(function() {"
        "  class Request {"
        "    constructor(input, init) {"
        "      if (typeof input === 'string') {"
        "        this.url = input;"
        "      } else if (input && input.url) {"
        "        this.url = input.url;"
        "        this.method = input.method || 'GET';"
        "        this.headers = input.headers || new Headers();"
        "        this.body = input.body || null;"
        "      } else {"
        "        this.url = '/';"
        "      }"
        "      if (init) {"
        "        this.method = init.method || this.method || 'GET';"
        "        this.headers = new Headers(init.headers || this.headers);"
        "        this.body = init.body || this.body || null;"
        "      } else {"
        "        this.method = this.method || 'GET';"
        "        this.headers = this.headers || new Headers();"
        "        this.body = this.body || null;"
        "      }"
        "    }"
        "  }"
        "  globalThis.Request = Request;"
        "})();";
    
    result = JS_Eval(ctx, request_polyfill, strlen(request_polyfill), "<request-polyfill>", JS_EVAL_TYPE_GLOBAL);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[WARN] Failed to add Request polyfill: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
    } else {
        JS_FreeValue(ctx, result);
    }
}

// C callback for JS console: __bunbase_log(level, message). Sends log to host via send_log.
static JSValue js_bunbase_log(JSContext *ctx, JSValueConst this_val, int argc, JSValueConst *argv) {
    const char *level = "info";
    const char *message = "";
    const char *level_to_free = NULL;
    const char *message_to_free = NULL;
    if (argc >= 2) {
        level_to_free = JS_ToCString(ctx, argv[0]);
        message_to_free = JS_ToCString(ctx, argv[1]);
        level = level_to_free ? level_to_free : "info";
        message = message_to_free ? message_to_free : "";
    } else if (argc >= 1) {
        message_to_free = JS_ToCString(ctx, argv[0]);
        message = message_to_free ? message_to_free : "";
    }
    send_log(current_invoke_id[0] ? current_invoke_id : "bundle", level, message);
    if (level_to_free) JS_FreeCString(ctx, level_to_free);
    if (message_to_free) JS_FreeCString(ctx, message_to_free);
    return JS_UNDEFINED;
}

/* Helper: stringify multiple args for console.log("a", 1) -> "a 1" */
static void add_console_override(JSContext *ctx) {
    JSValue global = JS_GetGlobalObject(ctx);
    JSValue log_fn = JS_NewCFunction(ctx, js_bunbase_log, "__bunbase_log", 2);
    JS_SetPropertyStr(ctx, global, "__bunbase_log", log_fn);

    const char *console_inject =
        "(function(){"
        "  function stringifyArgs(args){"
        "    if (!args || args.length === 0) return '';"
        "    try {"
        "      return Array.from(args).map(function(x){"
        "        if (x === null) return 'null';"
        "        if (typeof x === 'object') return JSON.stringify(x);"
        "        return String(x);"
        "      }).join(' ');"
        "    } catch(e) { return String(args[0]); }"
        "  }"
        "  globalThis.console = {"
        "    log: function(){ __bunbase_log('info', stringifyArgs(arguments)); },"
        "    info: function(){ __bunbase_log('info', stringifyArgs(arguments)); },"
        "    warn: function(){ __bunbase_log('warn', stringifyArgs(arguments)); },"
        "    error: function(){ __bunbase_log('error', stringifyArgs(arguments)); },"
        "    debug: function(){ __bunbase_log('debug', stringifyArgs(arguments)); }"
        "  };"
        "})();";
    JSValue result = JS_Eval(ctx, console_inject, strlen(console_inject), "<console-override>", JS_EVAL_TYPE_GLOBAL);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[WARN] Failed to add console override: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
    } else {
        JS_FreeValue(ctx, result);
    }
    JS_FreeValue(ctx, global);
}

// Enforce resource limits
static void enforce_resource_limits(void) {
    if (caps.max_memory > 0) {
        struct rlimit rlim;
        // Get current limit first
        if (getrlimit(RLIMIT_AS, &rlim) == 0) {
            // Set new limit, but don't exceed the hard limit
            rlim_t new_limit = caps.max_memory;
            if (new_limit > rlim.rlim_max) {
                new_limit = rlim.rlim_max;
            }
            rlim.rlim_cur = new_limit;
            // On macOS, RLIMIT_AS might not be supported, try RLIMIT_RSS as fallback
            if (setrlimit(RLIMIT_AS, &rlim) != 0) {
                // Try RLIMIT_RSS on macOS (though it's deprecated, it might work)
                #ifdef __APPLE__
                if (setrlimit(RLIMIT_RSS, &rlim) != 0) {
                    // Silently fail - memory limits are best-effort on macOS
                    // fprintf(stderr, "[WARN] Failed to set memory limit: %s (this is expected on macOS)\n", strerror(errno));
                }
                #else
                fprintf(stderr, "[WARN] Failed to set memory limit: %s\n", strerror(errno));
                #endif
            }
        }
    }
    
    if (caps.max_fds > 0) {
        struct rlimit rlim;
        // Get current limit first
        if (getrlimit(RLIMIT_NOFILE, &rlim) == 0) {
            rlim_t new_limit = caps.max_fds;
            if (new_limit > rlim.rlim_max) {
                new_limit = rlim.rlim_max;
            }
            rlim.rlim_cur = new_limit;
            if (setrlimit(RLIMIT_NOFILE, &rlim) != 0) {
                fprintf(stderr, "[WARN] Failed to set FD limit: %s\n", strerror(errno));
            }
        }
    }
}

// Load JavaScript bundle and extract handler
static int load_bundle(const char *path) {
    FILE *f = fopen(path, "r");
    if (!f) {
        fprintf(stderr, "[ERROR] Failed to open bundle: %s\n", path);
        return -1;
    }
    
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);
    
    if (size > MAX_BUNDLE_SIZE) {
        fprintf(stderr, "[ERROR] Bundle too large: %ld bytes\n", size);
        fclose(f);
        return -1;
    }
    
    char *code = malloc(size + 1);
    if (!code) {
        fprintf(stderr, "[ERROR] Failed to allocate memory for bundle\n");
        fclose(f);
        return -1;
    }
    
    size_t read = fread(code, 1, size, f);
    fclose(f);
    code[read] = '\0';
    
    // Properly load ES module in QuickJS:
    // 1. Compile the module
    // 2. Set import.meta
    // 3. Execute the module function
    // 4. Get module namespace
    // 5. Extract exports
    
    // Step 1: Compile the module
    JSValue module_func = JS_Eval(ctx, code, size, path, JS_EVAL_TYPE_MODULE | JS_EVAL_FLAG_COMPILE_ONLY);
    if (JS_IsException(module_func)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[ERROR] Failed to compile bundle: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        free(code);
        return -1;
    }
    
    // Step 2: Resolve the module (required before execution)
    if (JS_ResolveModule(ctx, module_func) < 0) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[ERROR] Failed to resolve module: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        JS_FreeValue(ctx, module_func);
        free(code);
        return -1;
    }
    
    // Step 3: Set import.meta (required for modules)
    if (js_module_set_import_meta(ctx, module_func, 1, 1) < 0) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[ERROR] Failed to set import.meta: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        JS_FreeValue(ctx, module_func);
        free(code);
        return -1;
    }
    
    // Step 4: Get the module definition (keep a reference to module_func)
    JSModuleDef *m = JS_VALUE_GET_PTR(module_func);
    if (!m) {
        fprintf(stderr, "[ERROR] Failed to get module definition\n");
        JS_FreeValue(ctx, module_func);
        free(code);
        return -1;
    }
    
    // Step 5: Execute the module function (duplicate to keep reference)
    JSValue result = JS_EvalFunction(ctx, JS_DupValue(ctx, module_func));
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[ERROR] Failed to execute bundle: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        JS_FreeValue(ctx, module_func);
        free(code);
        return -1;
    }
    
    // Step 6: Await the result if it's a promise (modules can be async)
    // Note: js_std_await always handles promises, even if result is not a promise
    result = js_std_await(ctx, result);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[ERROR] Module execution failed: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        JS_FreeValue(ctx, module_func);
        free(code);
        return -1;
    }
    
    // Step 7: Get the module namespace (m is still valid)
    JSValue module_ns = JS_GetModuleNamespace(ctx, m);
    JS_FreeValue(ctx, module_func);
    JS_FreeValue(ctx, result);
    
    if (JS_IsException(module_ns)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        fprintf(stderr, "[ERROR] Failed to get module namespace: %s\n", error);
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        free(code);
        return -1;
    }
    
    // Step 5: Get default export from module namespace
    JSValue default_export = JS_GetPropertyStr(ctx, module_ns, "default");
    
    if (JS_IsFunction(ctx, default_export)) {
        handler_func = default_export;
        JS_FreeValue(ctx, module_ns);
        free(code);
        return 0;
    }
    
    // Try named 'handler' export
    if (!JS_IsUndefined(default_export) && !JS_IsException(default_export)) {
        JS_FreeValue(ctx, default_export);
    }
    
    JSValue handler = JS_GetPropertyStr(ctx, module_ns, "handler");
    if (JS_IsFunction(ctx, handler)) {
        handler_func = handler;
        JS_FreeValue(ctx, module_ns);
        free(code);
        return 0;
    }
    
    // Cleanup and report error
    if (!JS_IsUndefined(handler) && !JS_IsException(handler)) {
        JS_FreeValue(ctx, handler);
    }
    JS_FreeValue(ctx, module_ns);
    fprintf(stderr, "[ERROR] No handler function found (expected default export or 'handler')\n");
    free(code);
    return -1;
}

// Execute handler function with request
static int execute_handler(const char *invoke_id, const char *method, const char *path,
                          const char *headers_json, const char *query_json, const char *body_base64) {
    if (JS_IsUndefined(handler_func)) {
        send_error(invoke_id, "Handler not loaded", "HANDLER_NOT_LOADED");
        return -1;
    }
    current_invoke_id[0] = '\0';
    if (invoke_id) {
        strncpy(current_invoke_id, invoke_id, sizeof(current_invoke_id) - 1);
        current_invoke_id[sizeof(current_invoke_id) - 1] = '\0';
    }

    // Create Request object with proper URL
    char request_code[8192];
    // Build full URL with path (query params will be added via searchParams)
    char full_url[1024];
    snprintf(full_url, sizeof(full_url), "%s", path ? path : "/");
    
    snprintf(request_code, sizeof(request_code),
             "(function() {"
             "  const urlStr = '%s';"
             "  const url = new URL(urlStr, 'http://localhost');"
             "  const query = %s;"
             "  for (const [k, v] of Object.entries(query)) { url.searchParams.set(k, v); }"
             "  const headers = %s;"
             "  const body = '%s' ? atob('%s') : null;"
             "  const req = new Request(url.toString(), { method: '%s', headers: headers, body: body });"
             "  return req;"
             "})()",
             full_url,
             query_json ? query_json : "{}",
             headers_json ? headers_json : "{}",
             body_base64 ? body_base64 : "",
             body_base64 ? body_base64 : "",
             method ? method : "GET");
    
    JSValue request_val = JS_Eval(ctx, request_code, strlen(request_code), "<request>", JS_EVAL_TYPE_GLOBAL);
    if (JS_IsException(request_val)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        send_error(invoke_id, error, "REQUEST_CREATION_ERROR");
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        current_invoke_id[0] = '\0';
        return -1;
    }
    
    // Call handler (may return a Promise)
    JSValue result = JS_Call(ctx, handler_func, JS_UNDEFINED, 1, &request_val);
    JS_FreeValue(ctx, request_val);
    
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        send_error(invoke_id, error, "HANDLER_ERROR");
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        current_invoke_id[0] = '\0';
        return -1;
    }
    
    // Await the result if it's a Promise (handler is async)
    result = js_std_await(ctx, result);
    if (JS_IsException(result)) {
        JSValue exception = JS_GetException(ctx);
        const char *error = JS_ToCString(ctx, exception);
        send_error(invoke_id, error, "HANDLER_ERROR");
        JS_FreeCString(ctx, error);
        JS_FreeValue(ctx, exception);
        current_invoke_id[0] = '\0';
        return -1;
    }
    
    // Extract status, headers, body from Response object
    JSValue status_val = JS_GetPropertyStr(ctx, result, "status");
    JSValue headers_val = JS_GetPropertyStr(ctx, result, "headers");
    JSValue body_val = JS_GetPropertyStr(ctx, result, "body");
    
    int status = 200;
    if (!JS_IsUndefined(status_val)) {
        JS_ToInt32(ctx, &status, status_val);
    }
    
    // Convert headers to JSON
    // Headers object has a _headers property that contains the actual map
    const char *headers_str = "{}";
    char *headers_to_free = NULL;
    if (!JS_IsUndefined(headers_val)) {
        // Try to get the _headers property first (from our polyfill)
        JSValue headers_map = JS_GetPropertyStr(ctx, headers_val, "_headers");
        int free_headers_map = 0;
        if (JS_IsUndefined(headers_map)) {
            // Fallback to headers property or the object itself
            JS_FreeValue(ctx, headers_map);
            headers_map = JS_GetPropertyStr(ctx, headers_val, "headers");
            free_headers_map = 1;
            if (JS_IsUndefined(headers_map)) {
                JS_FreeValue(ctx, headers_map);
                headers_map = JS_DupValue(ctx, headers_val);
                free_headers_map = 1;
            }
        } else {
            free_headers_map = 1;
        }
        
        JSValue headers_json_val = JS_JSONStringify(ctx, headers_map, JS_UNDEFINED, JS_UNDEFINED);
        if (!JS_IsException(headers_json_val) && JS_IsString(headers_json_val)) {
            headers_str = JS_ToCString(ctx, headers_json_val);
            headers_to_free = (char *)headers_str;
        }
        JS_FreeValue(ctx, headers_json_val);
        if (free_headers_map) {
            JS_FreeValue(ctx, headers_map);
        }
    }
    
    // Get body from Response object and base64 encode it
    // Response.json() creates a Response with body property containing the JSON string
    const char *body_str = "";
    char *body_to_free = NULL;
    char encoded_body[16384] = "";
    const char *body_encoded = "";
    
    if (!JS_IsUndefined(body_val) && !JS_IsNull(body_val)) {
        if (JS_IsString(body_val)) {
            body_str = JS_ToCString(ctx, body_val);
            body_to_free = (char *)body_str;
            
            // Base64 encode the body string
            // Simple base64 encoding implementation
            const char base64_chars[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
            size_t len = strlen(body_str);
            size_t i = 0, j = 0;
            
            for (i = 0; i < len && j < sizeof(encoded_body) - 4; i += 3) {
                uint32_t b = (unsigned char)body_str[i] << 16;
                if (i + 1 < len) b |= (unsigned char)body_str[i + 1] << 8;
                if (i + 2 < len) b |= (unsigned char)body_str[i + 2];
                
                encoded_body[j++] = base64_chars[(b >> 18) & 0x3F];
                encoded_body[j++] = base64_chars[(b >> 12) & 0x3F];
                if (i + 1 < len) {
                    encoded_body[j++] = base64_chars[(b >> 6) & 0x3F];
                } else {
                    encoded_body[j++] = '=';
                }
                if (i + 2 < len) {
                    encoded_body[j++] = base64_chars[b & 0x3F];
                } else {
                    encoded_body[j++] = '=';
                }
            }
            encoded_body[j] = '\0';
            body_encoded = encoded_body;
        }
    }
    
    // Send response with base64-encoded body
    send_response(invoke_id, status, headers_str, body_encoded);
    
    // Free C strings
    if (headers_to_free) {
        JS_FreeCString(ctx, headers_to_free);
    }
    if (body_to_free) {
        JS_FreeCString(ctx, body_to_free);
    }
    
    JS_FreeValue(ctx, status_val);
    JS_FreeValue(ctx, headers_val);
    JS_FreeValue(ctx, body_val);
    JS_FreeValue(ctx, result);
    
    current_invoke_id[0] = '\0';
    return 0;
}

// Parse and process NDJSON messages from stdin
static void process_messages(void) {
    char *line = NULL;
    size_t len = 0;
    ssize_t read;
    
    while ((read = getline(&line, &len, stdin)) != -1) {
        if (read == 0 || line[0] == '\n') {
            continue;
        }
        
        // Remove newline
        if (line[read - 1] == '\n') {
            line[read - 1] = '\0';
            read--;
        }
        
        // Parse JSON message using QuickJS JSON parser
        JSValue msg_val = JS_ParseJSON(ctx, line, read, "<stdin>");
        if (JS_IsException(msg_val)) {
            JSValue exception = JS_GetException(ctx);
            const char *error = JS_ToCString(ctx, exception);
            fprintf(stderr, "[ERROR] Failed to parse message: %s\n", error);
            JS_FreeCString(ctx, error);
            JS_FreeValue(ctx, exception);
            free(line);
            line = NULL;
            len = 0;
            continue;
        }
        
        // Extract message type
        JSValue type_val = JS_GetPropertyStr(ctx, msg_val, "type");
        const char *type_str = JS_ToCString(ctx, type_val);
        
        if (type_str && strcmp(type_str, "invoke") == 0) {
            // Extract invoke ID
            JSValue id_val = JS_GetPropertyStr(ctx, msg_val, "id");
            const char *invoke_id = JS_ToCString(ctx, id_val);
            
            // Extract payload
            JSValue payload_val = JS_GetPropertyStr(ctx, msg_val, "payload");
            if (!JS_IsUndefined(payload_val)) {
                // Extract method, path, headers, query, body from payload
                JSValue method_val = JS_GetPropertyStr(ctx, payload_val, "method");
                JSValue path_val = JS_GetPropertyStr(ctx, payload_val, "path");
                JSValue headers_val = JS_GetPropertyStr(ctx, payload_val, "headers");
                JSValue query_val = JS_GetPropertyStr(ctx, payload_val, "query");
                JSValue body_val = JS_GetPropertyStr(ctx, payload_val, "body");
                
                // Convert to C strings
                const char *method = JS_ToCString(ctx, method_val);
                const char *path = JS_ToCString(ctx, path_val);
                
                // Convert headers and query to JSON strings
                JSValue headers_json_val = JS_JSONStringify(ctx, headers_val, JS_UNDEFINED, JS_UNDEFINED);
                JSValue query_json_val = JS_JSONStringify(ctx, query_val, JS_UNDEFINED, JS_UNDEFINED);
                const char *headers_json = JS_ToCString(ctx, headers_json_val);
                const char *query_json = JS_ToCString(ctx, query_json_val);
                const char *body_str = JS_ToCString(ctx, body_val);
                
                // Execute handler
                execute_handler(invoke_id ? invoke_id : "unknown",
                               method ? method : "GET",
                               path ? path : "/",
                               headers_json ? headers_json : "{}",
                               query_json ? query_json : "{}",
                               body_str ? body_str : "");
                
                // Free C strings
                if (invoke_id) JS_FreeCString(ctx, invoke_id);
                if (method) JS_FreeCString(ctx, method);
                if (path) JS_FreeCString(ctx, path);
                if (headers_json) JS_FreeCString(ctx, headers_json);
                if (query_json) JS_FreeCString(ctx, query_json);
                if (body_str) JS_FreeCString(ctx, body_str);
                
                // Free JS values
                JS_FreeValue(ctx, method_val);
                JS_FreeValue(ctx, path_val);
                JS_FreeValue(ctx, headers_val);
                JS_FreeValue(ctx, query_val);
                JS_FreeValue(ctx, body_val);
                JS_FreeValue(ctx, headers_json_val);
                JS_FreeValue(ctx, query_json_val);
            } else {
                send_error(invoke_id ? invoke_id : "unknown", "Missing payload in invoke message", "INVALID_MESSAGE");
            }
            
            if (invoke_id) JS_FreeCString(ctx, invoke_id);
            JS_FreeValue(ctx, id_val);
            JS_FreeValue(ctx, payload_val);
        }
        
        if (type_str) JS_FreeCString(ctx, type_str);
        JS_FreeValue(ctx, type_val);
        JS_FreeValue(ctx, msg_val);
        
        free(line);
        line = NULL;
        len = 0;
    }
    
    if (line) {
        free(line);
    }
}

int main(int argc, char **argv) {
    // Get worker ID and bundle path from environment
    const char *wid = getenv("WORKER_ID");
    if (wid) {
        strncpy(worker_id, wid, sizeof(worker_id) - 1);
    } else {
        snprintf(worker_id, sizeof(worker_id), "worker-%ld", (long)getpid());
    }
    
    bundle_path = getenv("BUNDLE_PATH");
    if (!bundle_path) {
        fprintf(stderr, "[ERROR] BUNDLE_PATH environment variable required\n");
        return 1;
    }
    
    // Setup capabilities
    setup_capabilities();
    
    // Enforce resource limits
    enforce_resource_limits();
    
    // Initialize QuickJS runtime
    rt = JS_NewRuntime();
    if (!rt) {
        fprintf(stderr, "[ERROR] Failed to create QuickJS runtime\n");
        return 1;
    }
    
    ctx = JS_NewContext(rt);
    if (!ctx) {
        fprintf(stderr, "[ERROR] Failed to create QuickJS context\n");
        JS_FreeRuntime(rt);
        return 1;
    }
    
    // Load standard library
    js_std_init_handlers(rt);
    js_std_add_helpers(ctx, argc, argv);
    
    // Add Web API polyfills (URL, Response, Request)
    add_base64_polyfills(ctx);
    add_web_apis(ctx);
    add_console_override(ctx);
    
    // Disable eval if not allowed
    if (!caps.allow_eval) {
        // Remove eval and Function from global scope
        JSValue global = JS_GetGlobalObject(ctx);
        JSAtom eval_atom = JS_NewAtom(ctx, "eval");
        JSAtom function_atom = JS_NewAtom(ctx, "Function");
        JS_DeleteProperty(ctx, global, eval_atom, 0);
        JS_DeleteProperty(ctx, global, function_atom, 0);
        JS_FreeAtom(ctx, eval_atom);
        JS_FreeAtom(ctx, function_atom);
        JS_FreeValue(ctx, global);
    }
    
    // Load bundle
    if (load_bundle(bundle_path) != 0) {
        send_error("bundle-load", "Failed to load bundle", "BUNDLE_LOAD_ERROR");
        JS_FreeContext(ctx);
        JS_FreeRuntime(rt);
        return 1;
    }
    
    // Send ready message
    send_ready();
    
    // Process messages from stdin
    process_messages();
    
    // Cleanup
    if (!JS_IsUndefined(handler_func)) {
        JS_FreeValue(ctx, handler_func);
    }
    JS_FreeContext(ctx);
    JS_FreeRuntime(rt);
    
    return 0;
}
