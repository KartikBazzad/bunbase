/**
 * Function Code Validation
 * Validates function code syntax and security
 */

/**
 * Validation result
 */
export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

/**
 * Validate function code syntax
 * For Bun runtime, we can use Bun's built-in TypeScript/JavaScript parser
 */
export async function validateCodeSyntax(
  code: string,
  runtime: string = "bun",
): Promise<ValidationResult> {
  const errors: string[] = [];
  const warnings: string[] = [];

  if (!code || code.trim().length === 0) {
    return {
      valid: false,
      errors: ["Function code cannot be empty"],
      warnings: [],
    };
  }

  // Check for required handler export
  if (!code.includes("export") || !code.includes("handler")) {
    errors.push(
      "Function must export a 'handler' function: export async function handler(req: Request): Promise<Response>",
    );
  }

  // Check for dangerous patterns
  const dangerousPatterns = [
    {
      pattern: /eval\s*\(/,
      message: "Use of 'eval' is not allowed for security reasons",
    },
    {
      pattern: /Function\s*\(/,
      message: "Use of 'Function' constructor is not allowed for security reasons",
    },
    {
      pattern: /require\s*\(/,
      message: "Use of 'require' is not recommended. Use ES modules instead.",
      isWarning: true,
    },
    {
      pattern: /process\.exit/,
      message: "Use of 'process.exit' is not allowed",
    },
    {
      pattern: /child_process/,
      message: "Use of 'child_process' is not allowed for security reasons",
    },
    {
      pattern: /fs\.writeFile|fs\.writeFileSync/,
      message: "Direct file system writes are restricted. Use storage API instead.",
      isWarning: true,
    },
  ];

  for (const { pattern, message, isWarning } of dangerousPatterns) {
    if (pattern.test(code)) {
      if (isWarning) {
        warnings.push(message);
      } else {
        errors.push(message);
      }
    }
  }

  // Try to validate syntax using Bun's Transpiler
  try {
    // Use Bun's Transpiler to check if code is valid
    // This will throw if the code has syntax errors
    const transpiler = new Bun.Transpiler({
      loader: "ts", // Treat as TypeScript
    });
    transpiler.transformSync(code);
  } catch (error: any) {
    // If transpilation fails, it's likely a syntax error
    errors.push(`Syntax error: ${error.message || "Invalid code syntax"}`);
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Validate function handler signature
 */
export function validateHandlerSignature(code: string): ValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // Check for handler export
  const hasHandlerExport =
    code.includes("export") &&
    (code.includes("function handler") ||
      code.includes("const handler") ||
      code.includes("handler =") ||
      code.match(/export\s+(async\s+)?function\s+handler/));

  if (!hasHandlerExport) {
    errors.push(
      "Function must export a handler. Expected: export async function handler(req: Request): Promise<Response>",
    );
  }

  // Check for Request parameter
  if (hasHandlerExport && !code.includes("Request")) {
    warnings.push(
      "Handler should accept a Request parameter: handler(req: Request)",
    );
  }

  // Check for Response return type
  if (hasHandlerExport && !code.includes("Response")) {
    warnings.push(
      "Handler should return a Response: Promise<Response>",
    );
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Comprehensive validation
 */
export async function validateFunctionCode(
  code: string,
  runtime: string = "bun",
): Promise<ValidationResult> {
  const syntaxResult = await validateCodeSyntax(code, runtime);
  const signatureResult = validateHandlerSignature(code);

  return {
    valid: syntaxResult.valid && signatureResult.valid,
    errors: [...syntaxResult.errors, ...signatureResult.errors],
    warnings: [...syntaxResult.warnings, ...signatureResult.warnings],
  };
}
