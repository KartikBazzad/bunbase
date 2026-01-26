/**
 * Interactive prompts utility
 */

import prompts from "prompts";

export async function promptEmail(): Promise<string> {
  const response = await prompts({
    type: "text",
    name: "email",
    message: "Email:",
    validate: (value: string) => {
      if (!value) return "Email is required";
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
        return "Please enter a valid email address";
      }
      return true;
    },
  });

  if (!response.email) {
    process.exit(1);
  }

  return response.email;
}

export async function promptPassword(): Promise<string> {
  const response = await prompts({
    type: "password",
    name: "password",
    message: "Password:",
    validate: (value: string) => {
      if (!value) return "Password is required";
      if (value.length < 6) return "Password must be at least 6 characters";
      return true;
    },
  });

  if (!response.password) {
    process.exit(1);
  }

  return response.password;
}

export async function promptSelect<T extends string>(
  message: string,
  choices: Array<{ title: string; value: T }>,
): Promise<T> {
  const response = await prompts({
    type: "select",
    name: "value",
    message,
    choices,
  });

  if (!response.value) {
    process.exit(1);
  }

  return response.value;
}

export async function promptConfirm(message: string): Promise<boolean> {
  const response = await prompts({
    type: "confirm",
    name: "value",
    message,
    initial: true,
  });

  return response.value ?? false;
}

export async function promptText(
  message: string,
  initial?: string,
  validate?: (value: string) => boolean | string,
): Promise<string> {
  const response = await prompts({
    type: "text",
    name: "value",
    message,
    initial,
    validate,
  });

  if (!response.value) {
    process.exit(1);
  }

  return response.value;
}
