/**
 * Function template generators
 */

import { writeFileSync, mkdirSync, existsSync } from "fs";
import { join } from "path";
import type { FunctionRuntime, FunctionType } from "@bunbase/server-sdk";

export interface TemplateOptions {
  name: string;
  runtime: FunctionRuntime;
  type: FunctionType;
  handler: string;
  path?: string;
  methods?: string[];
}

/**
 * Generate Node.js/Bun HTTP function template
 */
function generateNodeJSHTTPTemplate(options: TemplateOptions): string {
  return `/**
 * ${options.name} - HTTP Function
 * 
 * This function handles HTTP requests at ${options.path || `/${options.name}`}
 * Supported methods: ${options.methods?.join(", ") || "GET, POST"}
 */

export async function ${options.handler}(req: Request): Promise<Response> {
  const { method, url, headers } = req;
  
  // Handle different HTTP methods
  if (method === "GET") {
    return Response.json({ 
      message: "Hello from BunBase!",
      function: "${options.name}",
      timestamp: new Date().toISOString()
    });
  }
  
  if (method === "POST") {
    try {
      const body = await req.json();
      // Process request body
      return Response.json({ 
        success: true, 
        data: body,
        message: "Request processed successfully"
      });
    } catch (error) {
      return Response.json(
        { error: "Invalid JSON body" },
        { status: 400 }
      );
    }
  }
  
  if (method === "PUT" || method === "PATCH") {
    try {
      const body = await req.json();
      return Response.json({ 
        success: true, 
        data: body,
        message: "Resource updated"
      });
    } catch (error) {
      return Response.json(
        { error: "Invalid JSON body" },
        { status: 400 }
      );
    }
  }
  
  if (method === "DELETE") {
    return Response.json({ 
      success: true,
      message: "Resource deleted"
    });
  }
  
  return Response.json(
    { error: "Method not allowed" },
    { status: 405 }
  );
}
`;
}

/**
 * Generate Node.js/Bun callable function template
 */
function generateNodeJSCallableTemplate(options: TemplateOptions): string {
  return `/**
 * ${options.name} - Callable Function
 * 
 * This function can be called from client SDKs with automatic authentication.
 * The context parameter contains user and session information.
 */

export async function ${options.handler}(
  data: any,
  context: { user?: any; session?: any }
): Promise<any> {
  // Access authenticated user via context.user
  // Access session via context.session
  
  // Example: Check if user is authenticated
  if (!context.user) {
    throw new Error("Authentication required");
  }
  
  // Your function logic here
  return {
    success: true,
    message: "Function executed successfully",
    data,
    userId: context.user?.id,
    timestamp: new Date().toISOString()
  };
}
`;
}

/**
 * Generate Python HTTP function template
 */
function generatePythonHTTPTemplate(options: TemplateOptions): string {
  return `"""
${options.name} - HTTP Function

This function handles HTTP requests at ${options.path || `/${options.name}`}
Supported methods: ${options.methods?.join(", ") || "GET, POST"}
"""

def ${options.handler}(req):
    """
    Handler function for HTTP requests
    
    Args:
        req: Request object with method, body, headers, etc.
    
    Returns:
        dict: Response with statusCode and body
    """
    method = req.get("method", "GET")
    body = req.get("body", {})
    
    if method == "GET":
        return {
            "statusCode": 200,
            "body": {
                "message": "Hello from BunBase!",
                "function": "${options.name}",
                "timestamp": __import__("datetime").datetime.now().isoformat()
            }
        }
    
    if method == "POST":
        return {
            "statusCode": 200,
            "body": {
                "success": True,
                "data": body,
                "message": "Request processed successfully"
            }
        }
    
    if method in ["PUT", "PATCH"]:
        return {
            "statusCode": 200,
            "body": {
                "success": True,
                "data": body,
                "message": "Resource updated"
            }
        }
    
    if method == "DELETE":
        return {
            "statusCode": 200,
            "body": {
                "success": True,
                "message": "Resource deleted"
            }
        }
    
    return {
        "statusCode": 405,
        "body": {"error": "Method not allowed"}
    }
`;
}

/**
 * Generate Python callable function template
 */
function generatePythonCallableTemplate(options: TemplateOptions): string {
  return `"""
${options.name} - Callable Function

This function can be called from client SDKs with automatic authentication.
The context parameter contains user and session information.
"""

def ${options.handler}(data, context):
    """
    Handler function for callable requests
    
    Args:
        data: Data passed from client
        context: Context object with user and session info
    
    Returns:
        dict: Response data
    """
    # Access authenticated user via context.get("user")
    # Access session via context.get("session")
    
    # Example: Check if user is authenticated
    user = context.get("user")
    if not user:
        raise Exception("Authentication required")
    
    # Your function logic here
    return {
        "success": True,
        "message": "Function executed successfully",
        "data": data,
        "userId": user.get("id") if user else None,
        "timestamp": __import__("datetime").datetime.now().isoformat()
    }
`;
}

/**
 * Generate package.json for Node.js/Bun functions
 */
function generatePackageJson(name: string): string {
  return JSON.stringify(
    {
      name: name,
      version: "0.1.0",
      type: "module",
      main: "index.ts",
      scripts: {
        dev: "bun --watch index.ts",
        start: "bun index.ts",
      },
    },
    null,
    2,
  );
}

/**
 * Generate requirements.txt for Python functions
 */
function generateRequirementsTxt(name: string): string {
  return `# Python dependencies for ${name}
# Add your dependencies here
`;
}

/**
 * Generate README for function
 */
function generateReadme(options: TemplateOptions): string {
  return `# ${options.name}

${options.type === "http" ? "HTTP" : "Callable"} function running on ${options.runtime}.

## Handler

\`${options.handler}\`

${options.type === "http" ? `## Path

\`${options.path || `/${options.name}`}\`

## Methods

${options.methods?.join(", ") || "GET, POST"}` : "## Authentication

This function requires authentication. The user context is automatically provided."}

## Local Development

\`\`\`bash
# For Node.js/Bun functions
bun --watch index.ts

# For Python functions
python index.py
\`\`\`

## Deployment

\`\`\`bash
bunbase functions deploy ${options.name}
\`\`\`
`;
}

/**
 * Bootstrap a function with templates
 */
export function bootstrapFunction(
  projectRoot: string,
  options: TemplateOptions,
): void {
  const functionDir = join(projectRoot, "functions", options.name);

  // Create directory
  if (existsSync(functionDir)) {
    throw new Error(`Function directory already exists: ${functionDir}`);
  }
  mkdirSync(functionDir, { recursive: true });

  // Generate main handler file
  let handlerContent: string;
  let handlerExtension: string;

  if (options.runtime.startsWith("python")) {
    handlerExtension = ".py";
    if (options.type === "http") {
      handlerContent = generatePythonHTTPTemplate(options);
    } else {
      handlerContent = generatePythonCallableTemplate(options);
    }
  } else {
    // Node.js/Bun
    handlerExtension = ".ts";
    if (options.type === "http") {
      handlerContent = generateNodeJSHTTPTemplate(options);
    } else {
      handlerContent = generateNodeJSCallableTemplate(options);
    }
  }

  // Determine handler file name
  const handlerFile = options.handler.includes(".")
    ? options.handler.split(".")[0] + handlerExtension
    : options.handler + handlerExtension;

  writeFileSync(join(functionDir, handlerFile), handlerContent);

  // Generate additional files based on runtime
  if (options.runtime.startsWith("python")) {
    writeFileSync(
      join(functionDir, "requirements.txt"),
      generateRequirementsTxt(options.name),
    );
  } else {
    // Node.js/Bun
    writeFileSync(
      join(functionDir, "package.json"),
      generatePackageJson(options.name),
    );
  }

  // Generate README
  writeFileSync(join(functionDir, "README.md"), generateReadme(options));

  // Generate .gitignore if it doesn't exist
  const gitignorePath = join(functionDir, ".gitignore");
  if (!existsSync(gitignorePath)) {
    writeFileSync(
      gitignorePath,
      `node_modules/
*.log
.env
.DS_Store
`,
    );
  }
}
