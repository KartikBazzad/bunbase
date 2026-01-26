export class JSONUtils {
  static toJSON(data: unknown): Uint8Array {
    const json = JSON.stringify(data);
    return new TextEncoder().encode(json);
  }

  static fromJSON<T>(data: Uint8Array): T {
    const json = new TextDecoder().decode(data);
    return JSON.parse(json) as T;
  }
}
